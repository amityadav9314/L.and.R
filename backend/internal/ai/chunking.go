package ai

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/amityadav/landr/pkg/pb/learning"
)

const (
	ChunkSize      = 6000 // ~1500 tokens - larger chunks to reduce parallel API calls
	ChunkOverlap   = 300  // Overlap between chunks
	MaxRetries     = 3
	BaseRetryDelay = 2 * time.Second
)

// ChunkResult holds the result from processing a single chunk
type ChunkResult struct {
	Title      string
	Tags       []string
	Flashcards []*learning.Flashcard
	Error      error
	ChunkIndex int
}

// EstimateTokens estimates token count (roughly chars/4)
func EstimateTokens(text string) int {
	return len(text) / 4
}

// SplitIntoChunks splits text into overlapping chunks
func SplitIntoChunks(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0
	maxChunks := 20 // Safety limit to prevent runaway

	for start < len(text) && len(chunks) < maxChunks {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at sentence boundary
		if end < len(text) {
			// Look for sentence end in last 100 chars
			searchStart := end - 100
			if searchStart < start {
				searchStart = start
			}
			chunk := text[searchStart:end]

			// Find last sentence boundary
			lastPeriod := strings.LastIndex(chunk, ". ")
			if lastPeriod > 0 {
				end = searchStart + lastPeriod + 2
			}
		}

		chunks = append(chunks, strings.TrimSpace(text[start:end]))

		// Ensure we always advance to prevent infinite loop
		newStart := end - overlap
		if newStart <= start {
			newStart = start + chunkSize/2 // Force advancement
		}
		start = newStart
	}

	log.Printf("[Chunker] Split text into %d chunks", len(chunks))
	return chunks
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		// Check if retryable error
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "429") ||
			strings.Contains(errStr, "500") ||
			strings.Contains(errStr, "502") ||
			strings.Contains(errStr, "503") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "connection refused")

		if !isRetryable {
			log.Printf("[Retry.%s] Non-retryable error: %v", operation, err)
			return err
		}

		// Calculate delay with exponential backoff + jitter
		delay := time.Duration(math.Pow(2, float64(attempt))) * BaseRetryDelay
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		delay += jitter

		log.Printf("[Retry.%s] Attempt %d failed: %v. Retrying in %v...", operation, attempt+1, err, delay)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ProcessChunksSequential processes chunks one at a time to respect rate limits
func (c *Client) ProcessChunksParallel(ctx context.Context, chunks []string, existingTags []string) (string, []string, []*learning.Flashcard, error) {
	if len(chunks) == 0 {
		return "", nil, nil, fmt.Errorf("no chunks to process")
	}

	if len(chunks) == 1 {
		// Single chunk - process normally
		return c.GenerateFlashcards(chunks[0], existingTags)
	}

	log.Printf("[AI.Sequential] Processing %d chunks sequentially (rate limit: 8000 TPM)", len(chunks))

	var allFlashcards []*learning.Flashcard
	allTags := make(map[string]bool)
	var firstTitle string
	var errors []error

	// Process chunks one at a time with delay
	for i, chunk := range chunks {
		if ctx.Err() != nil {
			break
		}

		log.Printf("[AI.Sequential] Processing chunk %d/%d", i+1, len(chunks))

		var title string
		var tags []string
		var cards []*learning.Flashcard
		var err error

		// Retry with backoff for this chunk
		err = RetryWithBackoff(ctx, fmt.Sprintf("Chunk_%d", i), func() error {
			title, tags, cards, err = c.GenerateFlashcards(chunk, existingTags)
			return err
		})

		if err != nil {
			log.Printf("[AI.Sequential] Chunk %d failed: %v", i+1, err)
			errors = append(errors, err)
		} else {
			if firstTitle == "" && title != "" {
				firstTitle = title
			}
			for _, tag := range tags {
				allTags[tag] = true
			}
			allFlashcards = append(allFlashcards, cards...)
			log.Printf("[AI.Sequential] Chunk %d completed: %d flashcards", i+1, len(cards))
		}

		// Wait 10 seconds between chunks to respect rate limit (8000 TPM)
		if i < len(chunks)-1 {
			log.Printf("[AI.Sequential] Waiting 10s for rate limit...")
			select {
			case <-ctx.Done():
				break
			case <-time.After(10 * time.Second):
			}
		}
	}

	// If all chunks failed, return error
	if len(errors) == len(chunks) {
		return "", nil, nil, fmt.Errorf("all chunks failed: %v", errors[0])
	}

	// Convert tags map to slice
	var tags []string
	for tag := range allTags {
		tags = append(tags, tag)
	}

	// Deduplicate flashcards
	dedupedCards := deduplicateFlashcards(allFlashcards)

	log.Printf("[AI.Sequential] Completed: %d unique flashcards from %d chunks", len(dedupedCards), len(chunks))
	return firstTitle, tags, dedupedCards, nil
}

// deduplicateFlashcards removes duplicate flashcards based on question similarity
func deduplicateFlashcards(cards []*learning.Flashcard) []*learning.Flashcard {
	seen := make(map[string]bool)
	var unique []*learning.Flashcard

	for _, card := range cards {
		// Normalize question for comparison
		key := strings.ToLower(strings.TrimSpace(card.Question))
		if len(key) > 50 {
			key = key[:50] // Use first 50 chars as key
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, card)
		}
	}

	return unique
}
