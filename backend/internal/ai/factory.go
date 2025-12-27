package ai

import (
	"fmt"

	"github.com/amityadav/landr/internal/ai/models"
)

// NewLLMProvider creates a provider instance based on the provider name.
// Returns nil and logs a fatal error if the provider is unsupported.
// Supported providers: "groq", "cerebras"
func NewLLMProvider(providerName, apiKey, modelID string) *BaseProvider {
	switch providerName {
	case "groq":
		return NewBaseProvider(ProviderConfig{
			Name:        "Groq",
			BaseURL:     "https://api.groq.com/openai/v1/chat/completions",
			APIKey:      apiKey,
			TextModel:   modelID,
			VisionModel: models.TaskVisionModel,
		})
	case "cerebras":
		return NewBaseProvider(ProviderConfig{
			Name:        "Cerebras",
			BaseURL:     "https://api.cerebras.ai/v1/chat/completions",
			APIKey:      apiKey,
			TextModel:   modelID,
			VisionModel: "", // Cerebras doesn't have vision model
		})
	default:
		// Fail fast: don't silently default to an unknown provider
		panic(fmt.Sprintf("unsupported AI provider: %s (supported: groq, cerebras)", providerName))
	}
}
