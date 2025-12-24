package ai

import (
	"log"
	"strings"
)

// ChunkConfig controls how content is split and processed
type ChunkConfig struct {
	MaxChunkChars int // Max characters per chunk (~4 chars = 1 token)
	OverlapChars  int // Overlap between chunks for context continuity
	MaxTotalChars int // Max total input chars before chunking kicks in
}

// DefaultChunkConfig returns sensible defaults for Groq's 8k token limit
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkChars: 20000, // ~5000 tokens per chunk, safe margin
		OverlapChars:  400,   // ~100 tokens overlap for context
		MaxTotalChars: 24000, // ~6000 tokens total before chunking
	}
}

// SplitIntoChunks splits large content into overlapping chunks
// Returns original content as single-element slice if under MaxTotalChars
func SplitIntoChunks(content string, config ChunkConfig) []string {
	if len(content) <= config.MaxTotalChars {
		return []string{content}
	}

	log.Printf("[Chunking] Content size %d chars exceeds %d, splitting into chunks...",
		len(content), config.MaxTotalChars)

	var chunks []string
	start := 0
	chunkNum := 0

	for start < len(content) {
		end := start + config.MaxChunkChars
		if end > len(content) {
			end = len(content)
		}

		// Try to break at a natural boundary (paragraph or sentence)
		if end < len(content) {
			// Look for paragraph break first
			if idx := strings.LastIndex(content[start:end], "\n\n"); idx > config.MaxChunkChars/2 {
				end = start + idx + 2
			} else if idx := strings.LastIndex(content[start:end], ". "); idx > config.MaxChunkChars/2 {
				// Fall back to sentence break
				end = start + idx + 2
			}
		}

		chunk := content[start:end]
		chunks = append(chunks, chunk)
		chunkNum++

		// Move start, accounting for overlap
		start = end - config.OverlapChars
		if start < 0 {
			start = 0
		}
		// Prevent infinite loop
		if start >= len(content) || end >= len(content) {
			break
		}
		// If we didn't move forward, force it
		if start <= end-config.MaxChunkChars {
			start = end
		}
	}

	log.Printf("[Chunking] Split into %d chunks", len(chunks))
	return chunks
}

// TruncateToLimit is a simple truncation for when chunking isn't appropriate
// (e.g., for agent tool results that must fit in one message)
func TruncateToLimit(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}
	log.Printf("[Chunking] Truncating from %d to %d chars", len(content), maxChars)
	return content[:maxChars] + "\n...[truncated]"
}

// EstimateTokens provides a rough token count (4 chars ≈ 1 token)
func EstimateTokens(content string) int {
	return len(content) / 4
}

// ChunkResult holds the result of processing a chunk
type ChunkResult struct {
	ChunkIndex int
	Result     string
	Error      error
}

// AggregateResults combines results from multiple chunks
// For flashcards: merges JSON arrays
// For summaries: concatenates with headers
func AggregateResults(results []ChunkResult, mode string) string {
	var validResults []string
	for _, r := range results {
		if r.Error == nil && r.Result != "" {
			validResults = append(validResults, r.Result)
		}
	}

	if len(validResults) == 0 {
		return ""
	}

	switch mode {
	case "summary":
		// For summaries, join with section markers
		var sb strings.Builder
		for i, r := range validResults {
			if i > 0 {
				sb.WriteString("\n\n---\n\n")
			}
			sb.WriteString(r)
		}
		return sb.String()
	default:
		// Default: just join
		return strings.Join(validResults, "\n")
	}
}
