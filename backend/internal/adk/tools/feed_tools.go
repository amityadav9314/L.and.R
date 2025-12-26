package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/prompts"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// ========================================
// get_user_preferences Tool
// ========================================

type GetPreferencesArgs struct {
	UserID string `json:"user_id"`
}

type GetPreferencesResult struct {
	Interests string `json:"interests"`
}

func NewGetPreferencesTool(s *store.PostgresStore) tool.Tool {
	handler := func(ctx tool.Context, args GetPreferencesArgs) (GetPreferencesResult, error) {
		log.Printf("[GetPreferencesTool] Fetching prefs for user: %s", args.UserID)
		if args.UserID == "" {
			return GetPreferencesResult{}, fmt.Errorf("missing user_id")
		}

		prefs, err := s.GetFeedPreferences(context.Background(), args.UserID)
		if err != nil {
			log.Printf("[GetPreferencesTool] Error fetching prefs: %v", err)
			return GetPreferencesResult{}, fmt.Errorf("failed to get prefs: %w", err)
		}

		log.Printf("[GetPreferencesTool] Found prefs - Enabled: %v, Length: %d", prefs.FeedEnabled, len(prefs.InterestPrompt))

		if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
			return GetPreferencesResult{Interests: "Feed is disabled or no interests set."}, nil
		}
		return GetPreferencesResult{Interests: fmt.Sprintf("User Interests: %s", prefs.InterestPrompt)}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "get_user_preferences",
		Description: prompts.ToolGetPreferencesDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create get_user_preferences tool: %v", err)
	}
	return t
}

// ========================================
// search_news Tool
// ========================================

type SearchNewsArgs struct {
	Queries []string `json:"queries"`
}

type SearchNewsResult struct {
	Articles string `json:"articles"`
}

func NewSearchNewsTool(tav *tavily.Client, serp *serpapi.Client) tool.Tool {
	handler := func(ctx tool.Context, args SearchNewsArgs) (SearchNewsResult, error) {
		const maxChars = 12000 // Increased limit for more context
		const maxArticles = 60 // Increased from 25 to allow more variety

		var allArticles []string
		totalChars := 0

		log.Printf("[SearchTool] Received %d queries: %v", len(args.Queries), args.Queries)

		for _, q := range args.Queries {
			if len(allArticles) >= maxArticles {
				break
			}

			// Rate Limiting: Wait between search queries
			log.Printf("[SearchTool] Waiting 5s before executing query: %s", q) // Reduced wait time slightly
			time.Sleep(5 * time.Second)

			// Tavily Search (News)
			if tav != nil && len(allArticles) < maxArticles {
				log.Printf("[SearchTool] Calling Tavily for query: %s", q)
				resp, err := tav.SearchWithOptions(q, tavily.SearchOptions{
					MaxResults: 10, // Increased from 3 to 10
					NewsOnly:   true,
					Days:       7,
				})
				if err == nil {
					log.Printf("[SearchTool] Tavily returned %d results", len(resp.Results))
					for _, r := range resp.Results {
						if len(allArticles) >= maxArticles || totalChars >= maxChars {
							break
						}
						// Limit content length to save tokens
						content := r.Content
						if len(content) > 300 {
							content = content[:300] + "..."
						}
						// Explicit instruction in Source field to guide LLM
						article := fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\nSource: TAVILY (Set provider='tavily')\n---", r.Title, r.URL, content)
						allArticles = append(allArticles, article)
						totalChars += len(article)
					}
				} else {
					log.Printf("[SearchTool] Tavily failed: %v", err)
				}
			}

			// SerpApi Search (Google News)
			if serp != nil && len(allArticles) < maxArticles && totalChars < maxChars {
				log.Printf("[SearchTool] Calling SerpApi (Google News) for query: %s", q)
				resp, err := serp.SearchNews(q)
				if err == nil {
					log.Printf("[SearchTool] SerpApi returned %d results", len(resp.Results))
					for i, r := range resp.Results {
						if i >= 10 || len(allArticles) >= maxArticles || totalChars >= maxChars { // Increased from 3 to 10
							break
						}
						snippet := r.Snippet
						if len(snippet) > 300 {
							snippet = snippet[:300] + "..."
						}
						// Explicit instruction in Source field
						article := fmt.Sprintf("Title: %s\nURL: %s\nSnippet: %s\nSource: GOOGLE (Set provider='google')\n---", r.Title, r.URL, snippet)
						allArticles = append(allArticles, article)
						totalChars += len(article)
					}
				} else {
					log.Printf("[SearchTool] SerpApi failed: %v", err)
				}
			}
		}

		if len(allArticles) == 0 {
			log.Printf("[SearchTool] No articles found across all queries.")
			return SearchNewsResult{Articles: "No articles found."}, nil
		}
		log.Printf("[SearchTool] Total unique articles collected: %d", len(allArticles))
		return SearchNewsResult{Articles: fmt.Sprintf("Found %d articles:\n\n%s", len(allArticles), strings.Join(allArticles, "\n\n"))}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "search_news",
		Description: prompts.ToolSearchNewsDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create search_news tool: %v", err)
	}
	return t
}

// ========================================
// store_articles Tool
// ========================================

type ArticleInput struct {
	Title    string  `json:"title"`
	URL      string  `json:"url"`
	Snippet  string  `json:"snippet"`
	Score    float64 `json:"score"`
	Provider string  `json:"provider"` // "google" or "tavily"
}

type StoreArticlesArgs struct {
	UserID   string         `json:"user_id"`
	Articles []ArticleInput `json:"articles"`
}

type StoreArticlesResult struct {
	Message string `json:"message"`
}

func NewStoreArticlesTool(s *store.PostgresStore) tool.Tool {
	handler := func(ctx tool.Context, args StoreArticlesArgs) (StoreArticlesResult, error) {
		log.Printf("[StoreArticlesTool] Called with user_id=%s, articles count=%d", args.UserID, len(args.Articles))

		if args.UserID == "" {
			log.Printf("[StoreArticlesTool] ERROR: missing user_id")
			return StoreArticlesResult{}, fmt.Errorf("missing user_id")
		}

		count := 0
		today := time.Now().Truncate(24 * time.Hour)

		for _, a := range args.Articles {
			// Validate provider to ensure it matches UI expectation
			provider := strings.ToLower(a.Provider)
			if provider != "google" && provider != "tavily" {
				provider = "google" // Default to google if unknown/empty to ensure visibility
			}

			// Clamp/Normalize score (Handle "900%" or 0-100 inputs)
			score := a.Score
			if score > 1.0 {
				if score <= 100.0 {
					score = score / 100.0
				} else {
					score = 1.0 // Cap at 1.0 for absurd values
				}
			}

			article := &store.DailyArticle{
				Title:          a.Title,
				URL:            a.URL,
				Snippet:        a.Snippet,
				RelevanceScore: score,
				SuggestedDate:  today,
				Provider:       provider,
			}
			if err := s.StoreDailyArticle(context.Background(), args.UserID, article); err == nil {
				count++
			}
		}
		return StoreArticlesResult{Message: fmt.Sprintf("Successfully stored %d articles.", count)}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "store_articles",
		Description: prompts.ToolStoreArticlesDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create store_articles tool: %v", err)
	}
	return t
}
