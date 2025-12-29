package model

import (
	"context"
	"fmt"
	"iter"
	"log"
	"strings"

	"github.com/amityadav/landr/internal/ai/models"
	"github.com/amityadav/landr/pkg/adk/model/cerebras"
	"github.com/amityadav/landr/pkg/adk/model/groq"
	adkmodel "google.golang.org/adk/model"
)

// FallbackModel wraps two models and falls back to the second on rate limits
type FallbackModel struct {
	primary   adkmodel.LLM
	fallback  adkmodel.LLM
	modelName string
}

// NewFallbackModel creates a model that tries Groq first, then Cerebras on 429
func NewFallbackModel(groqAPIKey, cerebrasAPIKey, groqModelName string) (*FallbackModel, error) {
	// Map Groq model name to Cerebras equivalent
	cerebrasModelName := mapGroqToCerebrasModel(groqModelName)

	// Create Groq model (primary)
	primaryModel, err := groq.NewModel(groq.Config{
		APIKey:    groqAPIKey,
		ModelName: groqModelName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create primary (Groq) model: %w", err)
	}

	// Create Cerebras model (fallback)
	fallbackModel, err := cerebras.NewModel(cerebras.Config{
		APIKey:    cerebrasAPIKey,
		ModelName: cerebrasModelName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback (Cerebras) model: %w", err)
	}

	return &FallbackModel{
		primary:   primaryModel,
		fallback:  fallbackModel,
		modelName: groqModelName,
	}, nil
}

// mapGroqToCerebrasModel maps Groq model names to Cerebras equivalents
func mapGroqToCerebrasModel(groqModel string) string {
	switch groqModel {
	case models.ModelGroqGptOss120b:
		return models.ModelCerebrasGptOss120b
	case models.ModelGroqGptOss20b:
		return "gpt-oss-20b" // No Cerebras constant for this yet
	case models.ModelGroqLlama3_3_70b:
		return models.ModelCerebrasLlama3_3_70b
	case models.ModelGroqLlama3_1_8b:
		return models.ModelCerebrasLlama3_1_8b
	case models.ModelGroqQwen_32b:
		return models.ModelCerebrasQwen3_32b
	default:
		// Default to gpt-oss-120b if unknown
		return models.ModelCerebrasGptOss120b
	}
}

// Name returns the model name
func (m *FallbackModel) Name() string {
	return fmt.Sprintf("fallback-%s", m.modelName)
}

// GenerateContent tries primary model first, falls back to secondary on rate limit
func (m *FallbackModel) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		log.Printf("[FallbackModel] Trying primary model (Groq)...")

		// Try primary model
		primaryFailed := false
		var primaryError error

		for resp, err := range m.primary.GenerateContent(ctx, req, stream) {
			if err != nil {
				// Check if it's a rate limit error
				if isRateLimitError(err) {
					log.Printf("[FallbackModel] Primary model rate limited: %v", err)
					primaryFailed = true
					primaryError = err
					break
				}
				// Other errors - propagate immediately
				yield(nil, err)
				return
			}
			// Success - yield response
			if !yield(resp, nil) {
				return
			}
		}

		// If primary succeeded, we're done
		if !primaryFailed {
			return
		}

		// Primary failed with rate limit - try fallback
		log.Printf("[FallbackModel] Switching to fallback model (Cerebras)...")

		for resp, err := range m.fallback.GenerateContent(ctx, req, stream) {
			if err != nil {
				// If fallback also fails, return both errors
				yield(nil, fmt.Errorf("primary failed (%v), fallback failed (%v)", primaryError, err))
				return
			}
			if !yield(resp, nil) {
				return
			}
		}
	}
}

// isRateLimitError checks if an error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate_limit")
}
