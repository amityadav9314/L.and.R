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
	"strings"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model implements the adk.model.LLM interface via Groq/OpenAI-compatible APIs
type Model struct {
	apiKey    string
	baseURL   string
	modelName string
	client    *http.Client
}

// Config for creating a new Groq Model
type Config struct {
	APIKey    string
	BaseURL   string // Defaults to Groq endpoint
	ModelName string // Defaults to gpt-oss-120b
}

// Name returns the name of the model
func (m *Model) Name() string {
	return "groq-adapter"
}

// NewModel creates a new Groq model adapter from config
func NewModel(cfg Config) *Model {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.groq.com/openai/v1/chat/completions"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "openai/gpt-oss-120b"
	}
	return &Model{
		apiKey:    cfg.APIKey,
		baseURL:   cfg.BaseURL,
		modelName: cfg.ModelName,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

// --- Local types for OpenAI-compatible API (NOT exported) ---

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

// GenerateContent generates content from the model
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// 1. Convert ADK Request to Chat Messages
		var messages []chatMessage

		for _, content := range req.Contents {
			role := "user"
			if content.Role == "model" {
				role = "assistant"
			}
			if content.Role == "system" {
				role = "system"
			}

			text := ""
			for _, part := range content.Parts {
				if part.Text != "" {
					text += part.Text
				}
			}

			if text != "" {
				messages = append(messages, chatMessage{
					Role:    role,
					Content: text,
				})
			}
		}

		// Token limit safeguard using centralized chunking utility
		// Groq free tier: 8k tokens. Reserve ~2k for response, ~6k for input.
		const maxInputChars = 24000 // ~6000 tokens

		totalChars := 0
		for _, msg := range messages {
			totalChars += len(msg.Content)
		}

		if totalChars > maxInputChars {
			log.Printf("[GroqAdapter] WARNING: Input %d chars exceeds %d limit. Truncating messages...", totalChars, maxInputChars)
			// Distribute limit across messages
			maxPerMsg := maxInputChars / len(messages)
			for i := range messages {
				if len(messages[i].Content) > maxPerMsg {
					messages[i].Content = messages[i].Content[:maxPerMsg] + "\n...[truncated due to token limit]"
				}
			}
		}

		// 2. Prepare Request
		chatReq := chatRequest{
			Model:    m.modelName,
			Messages: messages,
		}

		// 3. Send Request
		respStr, err := m.sendRequest(chatReq)
		if err != nil {
			yield(nil, err)
			return
		}

		// 4. Convert Response to ADK format
		resp := &model.LLMResponse{
			Content: &genai.Content{
				Role: "model",
				Parts: []*genai.Part{
					genai.NewPartFromText(respStr),
				},
			},
			FinishReason: genai.FinishReasonStop,
		}

		yield(resp, nil)
	}
}

// sendRequest handles HTTP requests to the LLM API
func (m *Model) sendRequest(reqBody chatRequest) (string, error) {
	log.Printf("[GroqAdapter] Sending request to %s with model %s...", m.baseURL, m.modelName)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", m.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[GroqAdapter] Response status: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	log.Printf("[GroqAdapter] Success, response length: %d", len(content))
	return content, nil
}
