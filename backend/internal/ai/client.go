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

const (
	TextModel   = "openai/gpt-oss-120b"
	VisionModel = "meta-llama/llama-4-scout-17b-16e-instruct"
)

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.groq.com/openai/v1/chat/completions",
		client: &http.Client{
			Timeout: 60 * time.Second, // Increased for vision processing
		},
	}
}

// Message types for API requests
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

// sendRequest is the common HTTP request handler (DRY principle)
func (c *Client) sendRequest(reqBody interface{}, operation string) (string, error) {
	log.Printf("[AI.%s] Sending request to Groq API...", operation)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("[AI.%s] Failed to marshal request: %v", operation, err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("[AI.%s] Failed to create HTTP request: %v", operation, err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("[AI.%s] HTTP request failed: %v", operation, err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[AI.%s] Received response with status: %d", operation, resp.StatusCode)

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[AI.%s] API error response: %s", operation, string(bodyBytes))
		return "", fmt.Errorf("api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		log.Printf("[AI.%s] Failed to decode response: %v", operation, err)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		log.Printf("[AI.%s] No choices returned in response", operation)
		return "", fmt.Errorf("no choices returned")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	log.Printf("[AI.%s] Response received, length: %d", operation, len(content))
	return content, nil
}

// ExtractTextFromImage uses Vision LLM to extract text from an image
func (c *Client) ExtractTextFromImage(base64Image string) (string, error) {
	log.Printf("[AI.OCR] Starting text extraction from image, base64 length: %d", len(base64Image))

	// Ensure proper data URL format
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
		Model: VisionModel,
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

	log.Printf("[AI.OCR] Using vision model: %s", VisionModel)

	extractedText, err := c.sendRequest(reqBody, "OCR")
	if err != nil {
		return "", fmt.Errorf("OCR extraction failed: %w", err)
	}

	log.Printf("[AI.OCR] Successfully extracted text, length: %d", len(extractedText))
	return extractedText, nil
}

// GenerateFlashcards sends the content to Groq and expects a JSON object with title, tags, and flashcards.
func (c *Client) GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error) {
	log.Printf("[AI.Flashcards] Starting generation, content length: %d", len(content))

	// Truncate content if too long to stay within 8k TPM limit
	// 24000 chars ≈ 6000 tokens, leaving headroom for prompt (~800 tokens)
	maxLen := 24000
	if len(content) > maxLen {
		log.Printf("[AI.Flashcards] Truncating content from %d to %d chars", len(content), maxLen)
		content = content[:maxLen]
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
		Model: TextModel,
		Messages: []interface{}{
			textMessage{Role: "user", Content: prompt},
		},
	}

	log.Printf("[AI.Flashcards] Using model: %s", TextModel)

	rawContent, err := c.sendRequest(reqBody, "Flashcards")
	if err != nil {
		return "", nil, nil, err
	}

	// Clean up potential markdown formatting
	rawContent = cleanJSON(rawContent)
	log.Printf("[AI.Flashcards] Cleaned JSON length: %d", len(rawContent))

	var result struct {
		Title      string                `json:"title"`
		Tags       []string              `json:"tags"`
		Flashcards []*learning.Flashcard `json:"flashcards"`
	}

	if err := json.Unmarshal([]byte(rawContent), &result); err != nil {
		log.Printf("[AI.Flashcards] Failed to parse JSON: %v", err)
		log.Printf("[AI.Flashcards] Content was: %s", rawContent)
		return "", nil, nil, fmt.Errorf("failed to parse json: %w. Content: %s", err, rawContent)
	}

	log.Printf("[AI.Flashcards] Successfully parsed: Title='%s', Tags=%d, Flashcards=%d",
		result.Title, len(result.Tags), len(result.Flashcards))
	return result.Title, result.Tags, result.Flashcards, nil
}

// GenerateSummary sends content to Groq and returns a concise summary.
func (c *Client) GenerateSummary(content string) (string, error) {
	log.Printf("[AI.Summary] Starting generation, content length: %d", len(content))

	// Truncate content if too long to stay within token limits
	maxLen := 25000
	if len(content) > maxLen {
		log.Printf("[AI.Summary] Truncating content from %d to %d", len(content), maxLen)
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
		Model: TextModel,
		Messages: []interface{}{
			textMessage{Role: "user", Content: prompt},
		},
	}

	log.Printf("[AI.Summary] Using model: %s", TextModel)

	summary, err := c.sendRequest(reqBody, "Summary")
	if err != nil {
		return "", err
	}

	log.Printf("[AI.Summary] Successfully generated, length: %d", len(summary))
	return summary, nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
