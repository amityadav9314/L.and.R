package model

import (
	"fmt"

	"github.com/amityadav/landr/pkg/adk/model/groq"
	adkmodel "google.golang.org/adk/model"
)

// NewModel creates an ADK model adapter based on provider name.
// Supported providers: "groq" (future: "cerebras", "openai", etc.)
//
// Example:
//
//	model, err := NewModel("groq", apiKey, "llama-3.3-70b-versatile")
//	if err != nil {
//	    return err
//	}
func NewModel(providerName, apiKey, modelID string) (adkmodel.LLM, error) {
	switch providerName {
	case "groq":
		model, err := groq.NewModel(groq.Config{
			APIKey:    apiKey,
			ModelName: modelID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create groq model: %w", err)
		}
		return model, nil
	default:
		return nil, fmt.Errorf("unsupported ADK model provider: %s (supported: groq)", providerName)
	}
}
