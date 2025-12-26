package search

// Article represents a search result from any provider
type Article struct {
	Title    string
	URL      string
	Snippet  string
	Provider string // "tavily", "google", "bing", etc.
}

// SearchProvider is the interface all search providers must implement
type SearchProvider interface {
	// Name returns the provider identifier (e.g., "tavily", "google")
	Name() string

	// SearchNews searches for news articles
	SearchNews(query string, maxResults int) ([]Article, error)
}
