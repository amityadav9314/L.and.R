package ai

import (
	"fmt"
	"log"
	"strings"

	"github.com/amityadav/landr/pkg/pb/learning"
)

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

// GenerateCompletion uses primary provider
func (m *MultiProvider) GenerateCompletion(prompt string) (string, error) {
	return m.primary.GenerateCompletion(prompt)
}
