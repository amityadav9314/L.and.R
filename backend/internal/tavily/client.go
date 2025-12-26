package tavily

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/amityadav/landr/internal/search"
)

const apiURL = "https://api.tavily.com/search"

// Client is a Tavily Search API client
type Client struct {
	apiKey string
	client *http.Client
}

// NewClient creates a new Tavily API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SearchRequest represents the Tavily search request payload
type SearchRequest struct {
	Query          string   `json:"query"`
	APIKey         string   `json:"api_key"`
	SearchDepth    string   `json:"search_depth,omitempty"` // "basic" or "advanced"
	Topic          string   `json:"topic,omitempty"`        // "general" or "news"
	Days           int      `json:"days,omitempty"`         // Only for "news" topic - max age in days
	IncludeAnswer  bool     `json:"include_answer,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
}

// SearchResult represents a single search result from Tavily
type SearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"` // Snippet
	Score         float64 `json:"score"`
	RawContent    string  `json:"raw_content,omitempty"`
	PublishedDate string  `json:"published_date,omitempty"` // For news topic
}

// SearchResponse represents the Tavily search response
type SearchResponse struct {
	Query        string         `json:"query"`
	Answer       string         `json:"answer,omitempty"`
	Results      []SearchResult `json:"results"`
	ResponseTime float64        `json:"response_time"`
}

// SearchOptions allows configuring the search
type SearchOptions struct {
	MaxResults int
	Days       int  // Max age of articles (only for news topic)
	NewsOnly   bool // Use "news" topic for recent articles
}

// Search performs a search using the Tavily API
func (c *Client) Search(query string, maxResults int) (*SearchResponse, error) {
	return c.SearchWithOptions(query, SearchOptions{MaxResults: maxResults})
}

// SearchWithOptions performs a search with additional options
func (c *Client) SearchWithOptions(query string, opts SearchOptions) (*SearchResponse, error) {
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}

	reqBody := SearchRequest{
		Query:       query,
		APIKey:      c.apiKey,
		SearchDepth: "basic",
		MaxResults:  opts.MaxResults,
	}

	// For news/recent articles
	if opts.NewsOnly {
		reqBody.Topic = "news"
		if opts.Days > 0 {
			reqBody.Days = opts.Days
		} else {
			reqBody.Days = 3 // Default to last 3 days for freshness
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("[Tavily] Searching for: %q (max %d results, topic=%s, days=%d)", query, opts.MaxResults, reqBody.Topic, reqBody.Days)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[Tavily] Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %d %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("[Tavily] Found %d results for query: %s", len(searchResp.Results), query)
	return &searchResp, nil
}

// Name returns the provider identifier
func (c *Client) Name() string {
	return "tavily"
}

// SearchNews implements the SearchProvider interface
func (c *Client) SearchNews(query string, maxResults int) ([]search.Article, error) {
	resp, err := c.SearchWithOptions(query, SearchOptions{
		MaxResults: maxResults,
		NewsOnly:   true,
		Days:       7,
	})
	if err != nil {
		return nil, err
	}

	articles := make([]search.Article, len(resp.Results))
	for i, r := range resp.Results {
		articles[i] = search.Article{
			Title:    r.Title,
			URL:      r.URL,
			Snippet:  r.Content,
			Provider: "tavily",
		}
	}
	return articles, nil
}
