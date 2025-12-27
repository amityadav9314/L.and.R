package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"net/http"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model implements model.Model for Groq API with tool calling support
type Model struct {
	apiKey    string
	baseURL   string
	modelName string
	client    *http.Client
}

// Config for creating a Groq model
type Config struct {
	APIKey    string
	BaseURL   string
	ModelName string
}

// Name returns the name of the model
func (m *Model) Name() string {
	return "groq-adapter"
}

// NewModel creates a new Groq model adapter from config.
// Returns error if required fields (APIKey, ModelName) are missing.
func NewModel(cfg Config) (*Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("groq: APIKey is required")
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("groq: ModelName is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1/chat/completions"
	}

	return &Model{
		apiKey:    cfg.APIKey,
		baseURL:   baseURL,
		modelName: cfg.ModelName,
		client:    &http.Client{Timeout: 300 * time.Second},
	}, nil
}

// --- OpenAI-compatible API types ---

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []toolDef     `json:"tools,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type functionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// GenerateContent generates content from the model
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// 1. Convert ADK Request to Chat Messages
		var messages []chatMessage

		// Track tool call IDs to map responses back to calls
		// ADK doesn't persist IDs across turns easily, so we generate deterministic IDs based on index
		// toolCallIDs := make(map[int]string) // MsgIndex -> ID

		for _, content := range req.Contents {
			// Handle Tool Responses (ADK sends them as separate turns with FunctionResponse parts)
			isToolResponse := false
			for _, part := range content.Parts {
				if part.FunctionResponse != nil {
					isToolResponse = true
					break
				}
			}

			if isToolResponse {
				for _, part := range content.Parts {
					if part.FunctionResponse != nil {
						// We need a tool_call_id. Since ADK might not preserve it, we'll try to find it
						// or default to a generated one if we are lenient.
						// However, OpenAI is strict.
						// Strategy: Should have been stored from previous assistant message.
						// Simplify: Just send the response with role "tool".
						// For now, let's use a placeholder ID if missing, but ideally we match it.
						// Log inspection showed no ID in FunctionResponse.
						// We will generate a consistent ID for the PREVIOUS tool call and reuse it.

						// NOTE: This simple adapter assumes synchronous turn-by-turn.
						// Real solution requires tracking IDs.
						// For this fix, let's assume one tool call per turn or match by name.

						// Let's use the Name as ID suffix or look up a map if we had one.
						// Since we don't have the ID from ADK, we'll use a deterministic ID "call_<name>"
						// and ensure we sent that same ID in the Assistant message.

						jsonBytes, _ := json.Marshal(part.FunctionResponse.Response)
						messages = append(messages, chatMessage{
							Role:       "tool",
							Content:    string(jsonBytes),
							ToolCallID: fmt.Sprintf("call_%s", part.FunctionResponse.Name),
						})
					}
				}
				continue
			}

			role := "user"
			if content.Role == "model" {
				role = "assistant"
			}
			if content.Role == "system" {
				role = "system"
			}

			// Handle Tool Calls (Assistant requesting tools)
			var toolCalls []toolCall
			text := ""

			for _, part := range content.Parts {
				if part.Text != "" {
					text += part.Text
				}
				if part.FunctionCall != nil {
					// Generate a deterministic ID we can reference later
					id := fmt.Sprintf("call_%s", part.FunctionCall.Name)

					// Marshal args to JSON string
					argsBytes, _ := json.Marshal(part.FunctionCall.Args)

					toolCalls = append(toolCalls, toolCall{
						ID:   id,
						Type: "function",
						Function: functionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsBytes),
						},
					})
				}
			}

			if text != "" || len(toolCalls) > 0 {
				messages = append(messages, chatMessage{
					Role:      role,
					Content:   text,
					ToolCalls: toolCalls,
				})
			}
		}

		// 2. Convert ADK Tools to OpenAI format
		var tools []toolDef
		if req.Tools != nil {
			for name, t := range req.Tools {
				// Try to get description from tool if it implements the interface
				desc := ""
				if describer, ok := t.(interface{ Description() string }); ok {
					desc = describer.Description()
				}

				tools = append(tools, toolDef{
					Type: "function",
					Function: functionDef{
						Name:        name,
						Description: desc,
						Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
					},
				})
			}
			log.Printf("[GroqAdapter] Sending %d tools to LLM: %v", len(tools), toolNames(tools))
		}

		// 3. Token limit safeguard
		const maxInputChars = 24000
		totalChars := 0
		for _, msg := range messages {
			totalChars += len(msg.Content)
		}
		if totalChars > maxInputChars {
			log.Printf("[GroqAdapter] WARNING: Input %d chars exceeds %d limit. Truncating...", totalChars, maxInputChars)
			maxPerMsg := maxInputChars / len(messages)
			for i := range messages {
				if len(messages[i].Content) > maxPerMsg {
					messages[i].Content = messages[i].Content[:maxPerMsg] + "\n...[truncated]"
				}
			}
		}

		// 4. Prepare Request
		chatReq := chatRequest{
			Model:    m.modelName,
			Messages: messages,
			Tools:    tools,
		}

		// 5. Send Request
		respMsg, err := m.sendRequest(chatReq)
		if err != nil {
			yield(nil, err)
			return
		}

		// 6. Convert Response to ADK format
		resp := &model.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{},
			},
		}

		// Handle tool calls in response
		if len(respMsg.ToolCalls) > 0 {
			log.Printf("[GroqAdapter] LLM requested %d tool calls", len(respMsg.ToolCalls))
			for _, tc := range respMsg.ToolCalls {
				log.Printf("[GroqAdapter] Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments)

				// Parse arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					log.Printf("[GroqAdapter] Failed to parse tool arguments: %v", err)
					continue
				}

				resp.Content.Parts = append(resp.Content.Parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: tc.Function.Name,
						Args: args,
					},
				})
			}
		}

		// Handle text response
		if respMsg.Content != "" {
			resp.Content.Parts = append(resp.Content.Parts, genai.NewPartFromText(respMsg.Content))
		}

		yield(resp, nil)
	}
}

func toolNames(tools []toolDef) []string {
	var names []string
	for _, t := range tools {
		names = append(names, t.Function.Name)
	}
	return names
}

func (m *Model) sendRequest(reqBody chatRequest) (*chatMessage, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[GroqAdapter] Attempt %d/%d: Sending to %s with model %s...", attempt, maxRetries, m.baseURL, m.modelName)

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", m.baseURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

		// Rate Limiting
		waitTime := 15 * time.Second
		if attempt > 1 {
			waitTime = time.Duration(15*attempt) * time.Second // Exponential backoff
		}
		log.Printf("[GroqAdapter] Waiting %v before API call (Rate Limit Safety)...", waitTime)
		time.Sleep(waitTime)

		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Printf("[GroqAdapter] Request error (attempt %d): %v", attempt, lastErr)
			continue
		}

		log.Printf("[GroqAdapter] Response status: %d", resp.StatusCode)

		if resp.StatusCode == 429 {
			// Rate limited - wait longer and retry
			resp.Body.Close()
			log.Printf("[GroqAdapter] Rate limited, waiting 60s before retry...")
			time.Sleep(60 * time.Second)
			continue
		}

		if resp.StatusCode == 400 {
			// Bad request - check if it's tool_use_failed (sometimes transient)
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			errMsg := string(bodyBytes)
			log.Printf("[GroqAdapter] 400 Error: %s", errMsg)

			// If tool_use_failed, retry with longer wait
			if attempt < maxRetries {
				log.Printf("[GroqAdapter] Retrying after tool_use_failed...")
				time.Sleep(time.Duration(30*attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("api error after %d attempts: %d %s", maxRetries, resp.StatusCode, errMsg)
		}

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("api error: %d %s", resp.StatusCode, string(bodyBytes))
			if resp.StatusCode >= 500 {
				// Server error - retry
				log.Printf("[GroqAdapter] Server error (attempt %d): %v", attempt, lastErr)
				continue
			}
			return nil, lastErr
		}

		var chatResp chatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(chatResp.Choices) == 0 {
			return nil, fmt.Errorf("no choices returned")
		}

		return &chatResp.Choices[0].Message, nil
	}

	return nil, fmt.Errorf("all %d retry attempts failed: %v", maxRetries, lastErr)
}
