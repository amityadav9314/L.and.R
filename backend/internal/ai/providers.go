package ai

import (
	"bytes"
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
	OptimizeSearchQuery(userInterests string) (string, error) // Convert user interests to optimized search query
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

// Internal types for AI API communication
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"`
}

type textMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type visionMessage struct {
	Role    string          `json:"role"`
	Content []visionContent `json:"content"`
}

type visionContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message textMessage `json:"message"`
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
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

// OptimizeSearchQuery converts user interests into an optimized search query
func (p *BaseProvider) OptimizeSearchQuery(userInterests string) (string, error) {
	log.Printf("[%s.SearchQuery] Optimizing query for: %s", p.config.Name, userInterests)

	// Calculate date 5 days ago for filtering
	fiveDaysAgo := time.Now().AddDate(0, 0, -5).Format("2006-01-02")

	prompt := fmt.Sprintf(`You are a news intelligence agent.

INPUT:
- Keywords: %s
- Time window: last 5 days (after:%s)

TASK:
1. Generate an optimized Google Search query that:
   - Covers ALL keywords using OR logic where appropriate
   - Returns ONLY relevant news/articles
   - Filters results to the last 5 days using after:%s
   - Excludes junk content using negative keywords: -jobs -hiring -courses -tutorial -reddit -quora -youtube -sponsored -advertisement
   - Prioritizes credible sources (news sites, blogs, Substack, Medium)

2. Requirements:
   - Keep total query under 25 terms (Google limit)
   - Use quoted phrases for multi-word topics
   - Include "after:%s" for date filtering
   - Add negative keywords for noise removal

3. Output ONLY the final Google search query text. No explanation, no formatting.

CONSTRAINTS:
- Be aggressive in filtering noise
- Optimize for relevance over volume
- Focus on signal, not coverage`, userInterests, fiveDaysAgo, fiveDaysAgo, fiveDaysAgo)

	reqBody := chatRequest{
		Model: p.config.TextModel,
		Messages: []interface{}{
			textMessage{Role: "user", Content: prompt},
		},
	}

	query, err := p.sendRequest(reqBody, "SearchQuery")
	if err != nil {
		// Fallback to original interests if LLM fails
		log.Printf("[%s.SearchQuery] LLM failed, using original: %v", p.config.Name, err)
		return userInterests, nil
	}

	// Clean up the query
	query = strings.TrimSpace(query)
	query = strings.Trim(query, `"'`)
	log.Printf("[%s.SearchQuery] Optimized query: %s", p.config.Name, query)
	return query, nil
}

// MultiProvider distributes work across providers to avoid rate limits
// Flashcards -> provider[0], Summary -> provider[1] (or wraps around)
type MultiProvider struct {
	providers []Provider
	primary   Provider // Used for OCR (only Groq has vision)
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

// GenerateFlashcards uses provider[0] with fallback to others
func (m *MultiProvider) GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error) {
	// Try provider 0 first (Groq), then fall back to others
	for i, provider := range m.providers {
		log.Printf("[MultiProvider] Trying %s for flashcards (attempt %d/%d)...", provider.Name(), i+1, len(m.providers))
		title, tags, cards, err := provider.GenerateFlashcards(content, existingTags)
		if err == nil {
			log.Printf("[MultiProvider] %s generated %d flashcards", provider.Name(), len(cards))
			return title, tags, cards, nil
		}
		log.Printf("[MultiProvider] %s failed: %v", provider.Name(), err)
	}
	return "", nil, nil, fmt.Errorf("all providers failed for flashcards")
}

// GenerateSummary uses provider[1] with fallback (distributes load)
func (m *MultiProvider) GenerateSummary(content string) (string, error) {
	// Start with provider 1 if available (Cerebras), else use 0
	startIdx := 0
	if len(m.providers) > 1 {
		startIdx = 1 // Use second provider (Cerebras) for summary
	}

	// Try starting from startIdx, then wrap around
	for i := 0; i < len(m.providers); i++ {
		idx := (startIdx + i) % len(m.providers)
		provider := m.providers[idx]
		log.Printf("[MultiProvider] Trying %s for summary...", provider.Name())
		summary, err := provider.GenerateSummary(content)
		if err == nil {
			log.Printf("[MultiProvider] %s generated summary (length: %d)", provider.Name(), len(summary))
			return summary, nil
		}
		log.Printf("[MultiProvider] %s failed: %v", provider.Name(), err)
	}
	return "", fmt.Errorf("all providers failed for summary")
}

// ExtractTextFromImage uses primary provider (only Groq has vision)
func (m *MultiProvider) ExtractTextFromImage(base64Image string) (string, error) {
	return m.primary.ExtractTextFromImage(base64Image)
}

// OptimizeSearchQuery uses primary provider
func (m *MultiProvider) OptimizeSearchQuery(userInterests string) (string, error) {
	return m.primary.OptimizeSearchQuery(userInterests)
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
