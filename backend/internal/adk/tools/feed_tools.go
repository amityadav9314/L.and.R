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
)

// NewGetPreferencesTool creates a tool to fetch user preferences
func NewGetPreferencesTool(s *store.PostgresStore) tool.Tool {
	return &Simple{
		NameVal: "get_user_preferences",
		DescVal: prompts.ToolGetPreferencesDesc,
		Fn: func(args map[string]interface{}) (string, error) {
			userID, _ := args["user_id"].(string)
			// ... (rest is same)
			if userID == "" {
				return "", fmt.Errorf("missing user_id")
			}

			// We need a context to call store methods. ADK doesn't pass it in Fn signature (yet),
			// but usually logic should happen in Call(ctx).
			// Our Simple.Call receives ctx, but Simple.Fn signature is func(args).
			// For simplicity in this refactor, we'll use Background or TODO if ctx is not available in closure.
			// Ideally Simple struct should pass ctx to Fn.
			// Let's UPDATE Simple definition in simple.go to pass context?
			// Or just capture context? No, capturing context from creation time is bad.
			// I'll update Simple.Fn signature in next step if needed, or just use context.Background()
			// strictly for this step since explicit context handling involves interface change.
			// Actually, DB calls usually require context.
			// Use context.Background() for now since Fn signature in Simple is strictly `func(map) (string, error)`.
			// Wait, I can update Simple struct since I just created it.
			// Let's assume I stick to the current signature for now and use Background,
			// or better: let's update Simple to standard.
			// But for now, using Background for DB calls in tool closure is acceptable for this MVP refactor.
			ctx := context.Background()

			prefs, err := s.GetFeedPreferences(ctx, userID)
			if err != nil {
				return "", fmt.Errorf("failed to get prefs: %w", err)
			}
			if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
				return "Feed is disabled or no interests set.", nil
			}
			return fmt.Sprintf("User Interests: %s", prefs.InterestPrompt), nil
		},
	}
}

// NewSearchNewsTool creates a tool to search for news
func NewSearchNewsTool(tav *tavily.Client, serp *serpapi.Client) tool.Tool {
	return &Simple{
		NameVal: "search_news",
		DescVal: prompts.ToolSearchNewsDesc,
		Fn: func(args map[string]interface{}) (string, error) {
			queriesInterface, ok := args["queries"].([]interface{})
			if !ok {
				return "", fmt.Errorf("queries must be list of strings")
			}

			// Token limit safeguard: ~6000 chars ≈ 1500 tokens, leaving room for agent reasoning
			const maxChars = 6000
			const maxArticles = 25 // Cap total articles to prevent token overflow

			var allArticles []string
			totalChars := 0

			for _, qInt := range queriesInterface {
				if len(allArticles) >= maxArticles {
					break
				}

				// Rate Limiting: Wait between search queries
				log.Printf("[SearchTool] Waiting 15s before executing query...")
				time.Sleep(15 * time.Second)

				q, ok := qInt.(string)
				if !ok {
					continue
				}

				// Tavily Search (News) - limit to 3 per query to save tokens
				if tav != nil && len(allArticles) < maxArticles {
					resp, err := tav.SearchWithOptions(q, tavily.SearchOptions{
						MaxResults: 3,
						NewsOnly:   true,
						Days:       3,
					})
					if err == nil {
						for _, r := range resp.Results {
							if len(allArticles) >= maxArticles || totalChars >= maxChars {
								break
							}
							// Truncate content to ~200 chars
							content := r.Content
							if len(content) > 200 {
								content = content[:200] + "..."
							}
							article := fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\nSource: Tavily\n---", r.Title, r.URL, content)
							allArticles = append(allArticles, article)
							totalChars += len(article)
						}
					}
				}

				// SerpApi Search (Google News) - limit to 3 per query
				if serp != nil && len(allArticles) < maxArticles && totalChars < maxChars {
					resp, err := serp.SearchNews(q)
					if err == nil {
						for i, r := range resp.Results {
							if i >= 3 || len(allArticles) >= maxArticles || totalChars >= maxChars {
								break
							}
							snippet := r.Snippet
							if len(snippet) > 200 {
								snippet = snippet[:200] + "..."
							}
							article := fmt.Sprintf("Title: %s\nURL: %s\nSnippet: %s\nSource: GoogleNews\n---", r.Title, r.URL, snippet)
							allArticles = append(allArticles, article)
							totalChars += len(article)
						}
					}
				}
			}

			if len(allArticles) == 0 {
				return "No articles found.", nil
			}
			return fmt.Sprintf("Found %d articles:\n\n%s", len(allArticles), strings.Join(allArticles, "\n\n")), nil
		},
	}
}

// NewStoreArticlesTool creates a tool to store articles
func NewStoreArticlesTool(s *store.PostgresStore) tool.Tool {
	return &Simple{
		NameVal: "store_articles",
		DescVal: prompts.ToolStoreArticlesDesc,
		Fn: func(args map[string]interface{}) (string, error) {
			userID, _ := args["user_id"].(string)
			articlesInterface, ok := args["articles"].([]interface{})
			if !ok {
				return "", fmt.Errorf("missing articles list")
			}

			count := 0
			today := time.Now().Truncate(24 * time.Hour)
			ctx := context.Background() // See note above about context

			for _, aInt := range articlesInterface {
				aMap, ok := aInt.(map[string]interface{})
				if !ok {
					continue
				}

				title, _ := aMap["title"].(string)
				url, _ := aMap["url"].(string)
				snippet, _ := aMap["snippet"].(string)
				score, _ := aMap["score"].(float64)

				article := &store.DailyArticle{
					Title:          title,
					URL:            url,
					Snippet:        snippet,
					RelevanceScore: score,
					SuggestedDate:  today,
					Provider:       "agent",
				}
				if err := s.StoreDailyArticle(ctx, userID, article); err == nil {
					count++
				}
			}
			return fmt.Sprintf("Successfully stored %d articles.", count), nil
		},
	}
}
