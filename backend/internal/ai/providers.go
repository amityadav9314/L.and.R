package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/amityadav/landr/pkg/pb/learning"
)

// Provider defines the interface for AI providers
type Provider interface {
	Name() string
	GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error)
	GenerateSummary(content string) (string, error)
	ExtractTextFromImage(base64Image string) (string, error)
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Name          string
	BaseURL       string
	APIKey        string
	TextModel     string
	VisionModel   string
	MaxContentLen int
}

// BaseProvider implements common functionality for OpenAI-compatible APIs
type BaseProvider struct {
	config ProviderConfig
	client *http.Client
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(config ProviderConfig) *BaseProvider {
	if config.MaxContentLen == 0 {
		config.MaxContentLen = 24000 // Default: ~6000 tokens
	}
	return &BaseProvider{
		config: config,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *BaseProvider) Name() string {
	return p.config.Name
}

// sendRequest handles HTTP requests to the AI provider
func (p *BaseProvider) sendRequest(reqBody interface{}, operation string) (string, error) {
	log.Printf("[%s.%s] Sending request...", p.config.Name, operation)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[%s.%s] Response status: %d", p.config.Name, operation, resp.StatusCode)

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
	log.Printf("[%s.%s] Success, response length: %d", p.config.Name, operation, len(content))
	return content, nil
}

// GenerateFlashcards implements flashcard generation
func (p *BaseProvider) GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error) {
	// Truncate content to stay within token limits
	if len(content) > p.config.MaxContentLen {
		log.Printf("[%s.Flashcards] Truncating from %d to %d chars", p.config.Name, len(content), p.config.MaxContentLen)
		content = content[:p.config.MaxContentLen]
	}

	prompt := fmt.Sprintf(`You are a helpful assistant that creates flashcards from text.
Analyze the following text and create:
1. A short, descriptive Title for the material.
2. A list of 3-5 relevant Tags (categories).
3. 6 to 40 high-quality flashcards (Question and Answer pairs).

Existing tags you might reuse if relevant: %s

Return ONLY a raw JSON object with the following structure:
{
  "title": "String",
  "tags": ["String", "String"],
  "flashcards": [
    {"question": "String", "answer": "String"}
  ]
}
Do not include any markdown formatting (like json code blocks).
Do not include any other text.

Text:
%s`, strings.Join(existingTags, ", "), content)

	reqBody := chatRequest{
		Model: p.config.TextModel,
		Messages: []interface{}{
			textMessage{Role: "user", Content: prompt},
		},
	}

	rawContent, err := p.sendRequest(reqBody, "Flashcards")
	if err != nil {
		return "", nil, nil, err
	}

	rawContent = cleanJSON(rawContent)

	var result struct {
		Title      string                `json:"title"`
		Tags       []string              `json:"tags"`
		Flashcards []*learning.Flashcard `json:"flashcards"`
	}

	if err := json.Unmarshal([]byte(rawContent), &result); err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse json: %w", err)
	}

	log.Printf("[%s.Flashcards] Parsed: Title='%s', Tags=%d, Cards=%d",
		p.config.Name, result.Title, len(result.Tags), len(result.Flashcards))
	return result.Title, result.Tags, result.Flashcards, nil
}

// GenerateSummary implements summary generation
func (p *BaseProvider) GenerateSummary(content string) (string, error) {
	maxLen := 25000
	if len(content) > maxLen {
		log.Printf("[%s.Summary] Truncating from %d to %d", p.config.Name, len(content), maxLen)
		content = content[:maxLen]
	}

	prompt := fmt.Sprintf(`You are a helpful assistant that creates concise summaries for learning materials.
Create a clear, well-structured summary of the following text that helps a student review the key concepts.
The summary should:
- Be 5-8 paragraphs
- Highlight the main concepts and key points
- Be easy to scan and review quickly
- Use bullet points where appropriate

Return ONLY the summary text, no additional formatting or metadata.

Text:
%s`, content)

	reqBody := chatRequest{
		Model: p.config.TextModel,
		Messages: []interface{}{
			textMessage{Role: "user", Content: prompt},
		},
	}

	return p.sendRequest(reqBody, "Summary")
}

// ExtractTextFromImage implements OCR using vision model
func (p *BaseProvider) ExtractTextFromImage(base64Image string) (string, error) {
	if p.config.VisionModel == "" {
		return "", fmt.Errorf("vision model not configured for %s", p.config.Name)
	}

	imageDataURL := base64Image
	if !strings.HasPrefix(base64Image, "data:") {
		imageDataURL = "data:image/jpeg;base64," + base64Image
	}

	prompt := `Extract ALL text from this image exactly as written.
Maintain the original structure, headings, and formatting.
If there are diagrams or charts, describe them briefly in brackets like [Diagram: description].
If the image contains handwritten text, do your best to transcribe it accurately.
Return ONLY the extracted text, no commentary or additional formatting.`

	reqBody := chatRequest{
		Model: p.config.VisionModel,
		Messages: []interface{}{
			visionMessage{
				Role: "user",
				Content: []visionContent{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &imageURL{URL: imageDataURL}},
				},
			},
		},
	}

	return p.sendRequest(reqBody, "OCR")
}

// MultiProvider runs multiple providers in parallel and returns the first successful result
type MultiProvider struct {
	providers []Provider
	primary   Provider // Used for operations that can't be parallelized (OCR)
}

// NewMultiProvider creates a new multi-provider orchestrator
func NewMultiProvider(providers ...Provider) *MultiProvider {
	if len(providers) == 0 {
		panic("at least one provider required")
	}
	return &MultiProvider{
		providers: providers,
		primary:   providers[0],
	}
}

func (m *MultiProvider) Name() string {
	names := make([]string, len(m.providers))
	for i, p := range m.providers {
		names[i] = p.Name()
	}
	return "Multi[" + strings.Join(names, "+") + "]"
}

// Result holds the result from a provider goroutine
type providerResult struct {
	provider   string
	title      string
	tags       []string
	flashcards []*learning.Flashcard
	summary    string
	err        error
}

// GenerateFlashcards runs all providers in parallel, returns first success
func (m *MultiProvider) GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	results := make(chan providerResult, len(m.providers))

	// Launch all providers in parallel
	for _, p := range m.providers {
		go func(provider Provider) {
			log.Printf("[MultiProvider] Starting %s for flashcards...", provider.Name())
			title, tags, cards, err := provider.GenerateFlashcards(content, existingTags)
			results <- providerResult{
				provider:   provider.Name(),
				title:      title,
				tags:       tags,
				flashcards: cards,
				err:        err,
			}
		}(p)
	}

	// Collect results - return first success
	var lastErr error
	for i := 0; i < len(m.providers); i++ {
		select {
		case result := <-results:
			if result.err == nil {
				log.Printf("[MultiProvider] %s won the race with %d flashcards!", result.provider, len(result.flashcards))
				return result.title, result.tags, result.flashcards, nil
			}
			log.Printf("[MultiProvider] %s failed: %v", result.provider, result.err)
			lastErr = result.err
		case <-ctx.Done():
			return "", nil, nil, fmt.Errorf("all providers timed out")
		}
	}

	return "", nil, nil, fmt.Errorf("all providers failed: %w", lastErr)
}

// GenerateSummary runs all providers in parallel, returns first success
func (m *MultiProvider) GenerateSummary(content string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	results := make(chan providerResult, len(m.providers))

	// Launch all providers in parallel
	for _, p := range m.providers {
		go func(provider Provider) {
			log.Printf("[MultiProvider] Starting %s for summary...", provider.Name())
			summary, err := provider.GenerateSummary(content)
			results <- providerResult{
				provider: provider.Name(),
				summary:  summary,
				err:      err,
			}
		}(p)
	}

	// Collect results - return first success
	var lastErr error
	for i := 0; i < len(m.providers); i++ {
		select {
		case result := <-results:
			if result.err == nil {
				log.Printf("[MultiProvider] %s won the race for summary!", result.provider)
				return result.summary, nil
			}
			log.Printf("[MultiProvider] %s failed: %v", result.provider, result.err)
			lastErr = result.err
		case <-ctx.Done():
			return "", fmt.Errorf("all providers timed out")
		}
	}

	return "", fmt.Errorf("all providers failed: %w", lastErr)
}

// ExtractTextFromImage uses primary provider (not parallelized - usually only one has vision)
func (m *MultiProvider) ExtractTextFromImage(base64Image string) (string, error) {
	return m.primary.ExtractTextFromImage(base64Image)
}

// Convenience constructors for specific providers

// NewGroqProvider creates a Groq provider
func NewGroqProvider(apiKey string) *BaseProvider {
	return NewBaseProvider(ProviderConfig{
		Name:        "Groq",
		BaseURL:     "https://api.groq.com/openai/v1/chat/completions",
		APIKey:      apiKey,
		TextModel:   "openai/gpt-oss-120b",
		VisionModel: "meta-llama/llama-4-scout-17b-16e-instruct",
	})
}

// NewCerebrasProvider creates a Cerebras provider
func NewCerebrasProvider(apiKey string) *BaseProvider {
	return NewBaseProvider(ProviderConfig{
		Name:        "Cerebras",
		BaseURL:     "https://api.cerebras.ai/v1/chat/completions",
		APIKey:      apiKey,
		TextModel:   "gpt-oss-120b",
		VisionModel: "", // Cerebras doesn't have vision model
	})
}
