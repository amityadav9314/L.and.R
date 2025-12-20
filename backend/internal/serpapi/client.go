package serpapi

import (
	"fmt"
	"log"

	g "github.com/serpapi/google-search-results-golang"
)

// Client is a wrapper around the SerpApi search service
type Client struct {
	apiKey string
}

// SearchResult represents a single organic result from SerpApi
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
	Score   float64 // We'll mock a score since SerpApi doesn't provide a direct one like Tavily
}

// SearchResponse represents the relevant parts of the SerpApi response
type SearchResponse struct {
	Results []SearchResult
}

// NewClient creates a new SerpApiClient
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

// Search performs a Google search via SerpApi and returns organic results
func (c *Client) Search(query string) (*SearchResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SerpApi API key is not set")
	}

	parameter := map[string]string{
		"engine":        "google",
		"q":             query,
		"location":      "Austin, Texas, United States",
		"google_domain": "google.com",
		"gl":            "us",
		"hl":            "en",
	}

	log.Printf("[SerpApi] Searching for: %q", query)
	search := g.NewGoogleSearch(parameter, c.apiKey)
	results, err := search.GetJSON()
	if err != nil {
		return nil, fmt.Errorf("serpapi search failed: %w", err)
	}

	// Focus on organic_results node
	organicResults, ok := results["organic_results"].([]interface{})
	if !ok {
		log.Printf("[SerpApi] No organic_results found in response")
		return &SearchResponse{Results: []SearchResult{}}, nil
	}

	var resultsList []SearchResult
	for i, item := range organicResults {
		res, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		title, _ := res["title"].(string)
		link, _ := res["link"].(string)
		snippet, _ := res["snippet"].(string)

		if title == "" || link == "" {
			continue
		}

		// Calculate a simple mock score based on position (higher position = higher score)
		// Tavily scores are usually 0.0 - 1.0
		score := 1.0 - (float64(i) * 0.05)
		if score < 0.1 {
			score = 0.1
		}

		resultsList = append(resultsList, SearchResult{
			Title:   title,
			URL:     link,
			Snippet: snippet,
			Score:   score,
		})
	}

	log.Printf("[SerpApi] Found %d organic results", len(resultsList))
	return &SearchResponse{
		Results: resultsList,
	}, nil
}
