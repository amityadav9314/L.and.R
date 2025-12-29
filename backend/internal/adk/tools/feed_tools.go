package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/search"
	"github.com/amityadav/landr/internal/store"
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
	Interests    string `json:"interests"`
	EvalCriteria string `json:"eval_criteria"`
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

		log.Printf("[GetPreferencesTool] Found prefs - Enabled: %v, Interests: %d chars, EvalPrompt: %d chars",
			prefs.FeedEnabled, len(prefs.InterestPrompt), len(prefs.FeedEvalPrompt))

		if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
			return GetPreferencesResult{
				Interests:    "Feed is disabled or no interests set.",
				EvalCriteria: "",
			}, nil
		}

		evalCriteria := prefs.FeedEvalPrompt
		if evalCriteria == "" {
			evalCriteria = "Ensure the article is informative, relevant to their interests, and not clickbait."
		}

		return GetPreferencesResult{
			Interests:    prefs.InterestPrompt,
			EvalCriteria: evalCriteria,
		}, nil
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
// search_news Tool (using SearchProvider interface)
// ========================================

type SearchNewsArgs struct {
	Queries []string `json:"queries"`
}

type SearchNewsResult struct {
	Articles string `json:"articles"`
}

func NewSearchNewsTool(providers []search.SearchProvider) tool.Tool {
	handler := func(ctx tool.Context, args SearchNewsArgs) (SearchNewsResult, error) {
		const maxChars = 30000 // Increased to fit more articles
		const maxArticles = 60

		var allArticles []string
		totalChars := 0

		log.Printf("[SearchTool] Received %d queries: %v", len(args.Queries), args.Queries)
		log.Printf("[SearchTool] Using %d search providers", len(providers))

		for _, query := range args.Queries {
			if len(allArticles) >= maxArticles {
				break
			}

			// Rate limiting between queries
			log.Printf("[SearchTool] Waiting 5s before executing query: %s", query)
			time.Sleep(5 * time.Second)

			// Search across all registered providers
			for _, provider := range providers {
				if len(allArticles) >= maxArticles || totalChars >= maxChars {
					break
				}

				log.Printf("[SearchTool] Calling %s for query: %s", provider.Name(), query)
				articles, err := provider.SearchNews(query, 10)
				if err != nil {
					log.Printf("[SearchTool] %s failed: %v", provider.Name(), err)
					continue
				}

				log.Printf("[SearchTool] %s returned %d results", provider.Name(), len(articles))
				for _, a := range articles {
					if len(allArticles) >= maxArticles || totalChars >= maxChars {
						break
					}

					// Limit content length to save tokens
					content := a.Snippet
					if len(content) > 300 {
						content = content[:300] + "..."
					}

					// Format article with explicit provider instruction
					providerUpper := strings.ToUpper(a.Provider)
					article := fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\nSource: %s (Set provider='%s')\n---",
						a.Title, a.URL, content, providerUpper, a.Provider)

					allArticles = append(allArticles, article)
					totalChars += len(article)
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
// evaluate_urls_batch Tool
// ========================================

type URLInput struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Snippet  string `json:"snippet"`
	Provider string `json:"provider"`
}

type EvaluateURLsBatchArgs struct {
	URLs         []URLInput `json:"urls"`
	Interests    string     `json:"interests"`
	EvalCriteria string     `json:"eval_criteria"`
}

type URLScore struct {
	URL      string  `json:"url"`
	Title    string  `json:"title"`
	Snippet  string  `json:"snippet"`
	Provider string  `json:"provider"`
	Score    float64 `json:"score"`
}

type EvaluateURLsBatchResult struct {
	Scores []URLScore `json:"scores"`
}

// cleanURL removes query parameters to save tokens
func cleanURL(u string) string {
	if idx := strings.Index(u, "?"); idx != -1 {
		return u[:idx]
	}
	return u
}

// truncateSnippet limits snippet length to save tokens
func truncateSnippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func NewEvaluateURLsBatchTool(ai ai.Provider) tool.Tool {
	handler := func(ctx tool.Context, args EvaluateURLsBatchArgs) (EvaluateURLsBatchResult, error) {
		log.Printf("[EvaluateURLsBatchTool] Evaluating %d URLs in batches", len(args.URLs))

		if len(args.URLs) == 0 {
			return EvaluateURLsBatchResult{Scores: []URLScore{}}, nil
		}

		criteria := args.EvalCriteria
		if criteria == "" {
			criteria = "Ensure the article is informative, relevant to their interests, and not clickbait."
		}

		// Process in batches of 5 to avoid rate limits
		const batchSize = 5
		const delayBetweenBatches = 10 * time.Second

		var allScores []URLScore

		for i := 0; i < len(args.URLs); i += batchSize {
			end := i + batchSize
			if end > len(args.URLs) {
				end = len(args.URLs)
			}

			batch := args.URLs[i:end]
			batchNum := (i / batchSize) + 1
			totalBatches := (len(args.URLs) + batchSize - 1) / batchSize

			log.Printf("[EvaluateURLsBatchTool] Processing batch %d/%d (%d URLs)", batchNum, totalBatches, len(batch))

			// Build compact URL list - clean URLs and truncate snippets
			var urlList strings.Builder
			for j, u := range batch {
				cleanedURL := cleanURL(u.URL)
				shortSnippet := truncateSnippet(u.Snippet, 80)
				urlList.WriteString(fmt.Sprintf("%d. %s | %s\n", j+1, u.Title, cleanedURL))
				if shortSnippet != "" {
					urlList.WriteString(fmt.Sprintf("   %s\n", shortSnippet))
				}
			}

			prompt := fmt.Sprintf(prompts.URLBatchEvaluation,
				args.Interests,
				criteria,
				urlList.String())

			resp, err := ai.GenerateCompletion(prompt)
			if err != nil {
				log.Printf("[EvaluateURLsBatchTool] Batch %d failed: %v, using default scores", batchNum, err)
				// On error, assign default scores for this batch but keep full article data
				for _, u := range batch {
					allScores = append(allScores, URLScore{
						URL:      u.URL,
						Title:    u.Title,
						Snippet:  u.Snippet,
						Provider: u.Provider,
						Score:    0.5,
					})
				}
			} else {
				// Parse JSON response - only contains url and score
				var batchScores []struct {
					URL   string  `json:"url"`
					Score float64 `json:"score"`
				}
				cleanResp := strings.TrimSpace(resp)
				cleanResp = strings.TrimPrefix(cleanResp, "```json")
				cleanResp = strings.TrimPrefix(cleanResp, "```")
				cleanResp = strings.TrimSuffix(cleanResp, "```")
				cleanResp = strings.TrimSpace(cleanResp)

				if err := json.Unmarshal([]byte(cleanResp), &batchScores); err != nil {
					log.Printf("[EvaluateURLsBatchTool] Batch %d: Failed to parse JSON, using defaults", batchNum)
					for _, u := range batch {
						allScores = append(allScores, URLScore{
							URL:      u.URL,
							Title:    u.Title,
							Snippet:  u.Snippet,
							Provider: u.Provider,
							Score:    0.5,
						})
					}
				} else {
					// Map scores back to original URLs and preserve full article data
					scoreMap := make(map[string]float64)
					for _, s := range batchScores {
						scoreMap[cleanURL(s.URL)] = s.Score
					}
					for _, u := range batch {
						score, ok := scoreMap[cleanURL(u.URL)]
						if !ok {
							score = 0.5 // Default if not found
						}
						allScores = append(allScores, URLScore{
							URL:      u.URL,
							Title:    u.Title,
							Snippet:  u.Snippet,
							Provider: u.Provider,
							Score:    score,
						})
					}
				}
			}

			// Wait before next batch (except for last batch)
			if end < len(args.URLs) {
				log.Printf("[EvaluateURLsBatchTool] Waiting %v before next batch...", delayBetweenBatches)
				time.Sleep(delayBetweenBatches)
			}
		}

		log.Printf("[EvaluateURLsBatchTool] Successfully evaluated %d URLs", len(allScores))
		return EvaluateURLsBatchResult{Scores: allScores}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "evaluate_urls_batch",
		Description: prompts.ToolEvaluateURLsBatchDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create evaluate_urls_batch tool: %v", err)
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
