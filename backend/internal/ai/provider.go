package ai

import "github.com/amityadav/landr/pkg/pb/learning"

// Provider defines the interface for AI providers
type Provider interface {
	Name() string
	GenerateFlashcards(content string, existingTags []string) (string, []string, []*learning.Flashcard, error)
	GenerateSummary(content string) (string, error)
	ExtractTextFromImage(base64Image string) (string, error)
	OptimizeSearchQuery(userInterests string) (string, error)
	GenerateCompletion(prompt string) (string, error)
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
