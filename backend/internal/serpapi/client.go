package serpapi

import (
	"fmt"
	"log"

	"github.com/amityadav/landr/internal/search"
	g "github.com/serpapi/google-search-results-golang"
)

// Client is a wrapper around the SerpApi search service
type Client struct {
	apiKey string
}

// SearchResult represents a single organic result from SerpApi
type SearchResult struct {
	Title         string
	URL           string
	Snippet       string
	PositionScore float64 // Position-based score (1.0 for first result, decreasing). NOT a relevance score. Use -1 if unavailable.
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

		// Position-based score (NOT relevance). Top result = 1.0, decreasing by 0.05 per position.
		positionScore := 1.0 - (float64(i) * 0.05)
		if positionScore < 0.1 {
			positionScore = 0.1
		}

		resultsList = append(resultsList, SearchResult{
			Title:         title,
			URL:           link,
			Snippet:       snippet,
			PositionScore: positionScore,
		})
	}

	log.Printf("[SerpApi] Found %d organic results", len(resultsList))
	return &SearchResponse{
		Results: resultsList,
	}, nil
}

// SearchNewsRaw performs a Google News search via SerpApi and returns raw response
func (c *Client) SearchNewsRaw(query string) (*SearchResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("SerpApi API key is not set")
	}

	parameter := map[string]string{
		"engine": "google_news",
		"q":      query,
		"gl":     "us",
		"hl":     "en",
		"num":    "10", // Limit to 10 results per query as requested
	}

	log.Printf("[SerpApi] Searching News for: %q", query)
	search := g.NewGoogleSearch(parameter, c.apiKey)
	results, err := search.GetJSON()
	if err != nil {
		return nil, fmt.Errorf("serpapi news search failed: %w", err)
	}

	// Focus on news_results node
	newsResults, ok := results["news_results"].([]interface{})
	if !ok {
		log.Printf("[SerpApi] No news_results found in response")
		return &SearchResponse{Results: []SearchResult{}}, nil
	}

	var resultsList []SearchResult
	for i, item := range newsResults {
		res, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		title, _ := res["title"].(string)
		link, _ := res["link"].(string)
		snippet, _ := res["snippet"].(string)
		// Issue #7: Don't use date as snippet, leave empty if no snippet available

		if title == "" || link == "" {
			continue
		}

		// Position-based score (NOT relevance). Use -1 if you want to indicate "unavailable".
		positionScore := 1.0 - (float64(i) * 0.05)
		if positionScore < 0.1 {
			positionScore = 0.1
		}

		resultsList = append(resultsList, SearchResult{
			Title:         title,
			URL:           link,
			Snippet:       snippet,
			PositionScore: positionScore,
		})
	}

	log.Printf("[SerpApi] Found %d news results (truncating to 10)", len(resultsList))
	if len(resultsList) > 10 {
		resultsList = resultsList[:10]
	}
	return &SearchResponse{
		Results: resultsList,
	}, nil
}

// Name returns the provider identifier
func (c *Client) Name() string {
	return "google"
}

// SearchNews implements the SearchProvider interface (using Google News)
func (c *Client) SearchNews(query string, maxResults int) ([]search.Article, error) {
	resp, err := c.SearchNewsRaw(query)
	if err != nil {
		return nil, err
	}

	// Limit results
	results := resp.Results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	articles := make([]search.Article, len(results))
	for i, r := range results {
		articles[i] = search.Article{
			Title:    r.Title,
			URL:      r.URL,
			Snippet:  r.Snippet,
			Provider: "google",
		}
	}
	return articles, nil
}
