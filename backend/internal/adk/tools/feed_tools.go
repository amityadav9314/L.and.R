package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/scraper"
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
		const maxChars = 12000
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
// scrape_content Tool
// ========================================

type ScrapeContentArgs struct {
	URL string `json:"url"`
}

type ScrapeContentResult struct {
	Content string `json:"content"`
}

func NewScrapeContentTool(scraper *scraper.Scraper) tool.Tool {
	handler := func(ctx tool.Context, args ScrapeContentArgs) (ScrapeContentResult, error) {
		log.Printf("[ScrapeContentTool] Scraping URL: %s", args.URL)

		if args.URL == "" {
			return ScrapeContentResult{}, fmt.Errorf("missing url")
		}

		content, err := scraper.Scrape(args.URL)
		if err != nil {
			log.Printf("[ScrapeContentTool] Scraping failed for %s: %v", args.URL, err)
			return ScrapeContentResult{}, fmt.Errorf("scraping failed: %w", err)
		}

		log.Printf("[ScrapeContentTool] Successfully scraped %d chars from %s", len(content), args.URL)
		return ScrapeContentResult{Content: content}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "scrape_content",
		Description: prompts.ToolScrapeContentDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create scrape_content tool: %v", err)
	}
	return t
}

// ========================================
// summarize_content Tool
// ========================================

type SummarizeContentArgs struct {
	Content string `json:"content"`
}

type SummarizeContentResult struct {
	Summary string `json:"summary"`
}

func NewSummarizeContentTool(ai ai.Provider) tool.Tool {
	handler := func(ctx tool.Context, args SummarizeContentArgs) (SummarizeContentResult, error) {
		log.Printf("[SummarizeContentTool] Summarizing %d chars of content", len(args.Content))

		if args.Content == "" {
			return SummarizeContentResult{}, fmt.Errorf("missing content")
		}

		summary, err := ai.GenerateSummary(args.Content)
		if err != nil {
			log.Printf("[SummarizeContentTool] Summarization failed: %v", err)
			return SummarizeContentResult{}, fmt.Errorf("summarization failed: %w", err)
		}

		log.Printf("[SummarizeContentTool] Generated summary of %d chars", len(summary))
		return SummarizeContentResult{Summary: summary}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "summarize_content",
		Description: prompts.ToolSummarizeContentDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create summarize_content tool: %v", err)
	}
	return t
}

// ========================================
// evaluate_article Tool
// ========================================

type EvaluateArticleArgs struct {
	Summary      string `json:"summary"`
	Interests    string `json:"interests"`
	EvalCriteria string `json:"eval_criteria"`
}

type EvaluateArticleResult struct {
	Score float64 `json:"score"`
}

func NewEvaluateArticleTool(ai ai.Provider) tool.Tool {
	handler := func(ctx tool.Context, args EvaluateArticleArgs) (EvaluateArticleResult, error) {
		log.Printf("[EvaluateArticleTool] Evaluating article summary (%d chars)", len(args.Summary))

		if args.Summary == "" {
			return EvaluateArticleResult{Score: 0.0}, nil
		}

		criteria := args.EvalCriteria
		if criteria == "" {
			criteria = "Ensure the article is informative, relevant to their interests, and not clickbait."
		}

		prompt := fmt.Sprintf(prompts.ArticleEvaluation, args.Interests, criteria, args.Summary)

		resp, err := ai.GenerateCompletion(prompt)
		if err != nil {
			log.Printf("[EvaluateArticleTool] Evaluation failed: %v", err)
			return EvaluateArticleResult{Score: 0.0}, nil // Return 0 on error instead of failing
		}

		// Parse float score
		var score float64
		_, err = fmt.Sscanf(strings.TrimSpace(resp), "%f", &score)
		if err != nil {
			log.Printf("[EvaluateArticleTool] Failed to parse score from: %s", resp)
			return EvaluateArticleResult{Score: 0.0}, nil
		}

		// Clamp score to 0.0-1.0 range
		if score < 0.0 {
			score = 0.0
		} else if score > 1.0 {
			score = 1.0
		}

		log.Printf("[EvaluateArticleTool] Article scored: %.2f", score)
		return EvaluateArticleResult{Score: score}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "evaluate_article",
		Description: prompts.ToolEvaluateArticleDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create evaluate_article tool: %v", err)
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
