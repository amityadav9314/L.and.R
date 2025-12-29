package core

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
)

// Article represents a search result
type Article struct {
	URL      string
	Title    string
	Snippet  string
	Provider string
}

// ScoredArticle includes relevance score
type ScoredArticle struct {
	Article
	Score float64
}

// FeedGenerator handles the feed generation workflow
type FeedGenerator struct {
	store      *store.PostgresStore
	providers  []search.SearchProvider
	aiProvider ai.Provider
}

// NewFeedGenerator creates a new feed generator
func NewFeedGenerator(s *store.PostgresStore, providers []search.SearchProvider, ai ai.Provider) *FeedGenerator {
	return &FeedGenerator{
		store:      s,
		providers:  providers,
		aiProvider: ai,
	}
}

// GenerateFeed generates and stores daily feed for a user
func (g *FeedGenerator) GenerateFeed(ctx context.Context, userID, userEmail string) error {
	log.Printf("[FeedGenerator] Starting for user: %s (%s)", userEmail, userID)

	// 1. Get user preferences
	prefs, err := g.store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get preferences: %w", err)
	}

	if !prefs.FeedEnabled {
		log.Printf("[FeedGenerator] Feed disabled for user %s", userID)
		return nil
	}

	interests := strings.Split(prefs.InterestPrompt, ",")
	log.Printf("[FeedGenerator] User interests: %v", interests)

	// 2. Search for articles
	articles, err := g.searchArticles(ctx, interests)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	log.Printf("[FeedGenerator] Found %d unique articles", len(articles))

	// 3. Evaluate articles in batches
	scoredArticles, err := g.evaluateArticles(ctx, articles, prefs.InterestPrompt, prefs.FeedEvalPrompt)
	if err != nil {
		log.Printf("[FeedGenerator] Evaluation failed, using default scores: %v", err)
		// On failure, assign default score to all
		scoredArticles = make([]ScoredArticle, len(articles))
		for i, a := range articles {
			scoredArticles[i] = ScoredArticle{Article: a, Score: 0.5}
		}
	}

	// 4. Store ALL articles
	stored := 0
	today := time.Now().Truncate(24 * time.Hour)

	for _, sa := range scoredArticles {
		// Check if already exists
		exists, _ := g.store.ArticleURLExists(ctx, userID, sa.URL)
		if exists {
			log.Printf("[FeedGenerator] Skipping duplicate URL: %s", sa.URL)
			continue
		}

		article := &store.DailyArticle{
			Title:          sa.Title,
			URL:            sa.URL,
			Snippet:        sa.Snippet,
			RelevanceScore: sa.Score,
			SuggestedDate:  today,
			Provider:       sa.Provider,
		}
		if err := g.store.StoreDailyArticle(ctx, userID, article); err != nil {
			log.Printf("[FeedGenerator] Failed to store article %s: %v", sa.URL, err)
			continue
		}
		stored++
	}

	log.Printf("[FeedGenerator] Stored %d articles for user %s", stored, userID)
	return nil
}

// searchArticles searches all providers for articles
func (g *FeedGenerator) searchArticles(ctx context.Context, interests []string) ([]Article, error) {
	seen := make(map[string]bool)
	var articles []Article

	for _, interest := range interests {
		query := strings.TrimSpace(interest)
		if query == "" {
			continue
		}

		log.Printf("[FeedGenerator] Searching for: %s", query)

		for _, provider := range g.providers {
			results, err := provider.SearchNews(query, 10) // Get 10 results per query
			if err != nil {
				log.Printf("[FeedGenerator] Provider %s error: %v", provider.Name(), err)
				continue
			}

			for _, r := range results {
				// Clean URL (remove query params)
				url := cleanFeedURL(r.URL)
				if seen[url] {
					continue
				}
				seen[url] = true

				articles = append(articles, Article{
					URL:      r.URL,
					Title:    r.Title,
					Snippet:  truncateFeed(r.Snippet, 150),
					Provider: provider.Name(),
				})
			}
		}

		// Small delay between queries
		time.Sleep(2 * time.Second)
	}

	return articles, nil
}

// evaluateArticles evaluates articles in batches
func (g *FeedGenerator) evaluateArticles(ctx context.Context, articles []Article, interests, criteria string) ([]ScoredArticle, error) {
	const batchSize = 5
	const delayBetweenBatches = 10 * time.Second

	scored := make([]ScoredArticle, 0, len(articles))

	for i := 0; i < len(articles); i += batchSize {
		end := i + batchSize
		if end > len(articles) {
			end = len(articles)
		}
		batch := articles[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (len(articles) + batchSize - 1) / batchSize

		log.Printf("[FeedGenerator] Evaluating batch %d/%d (%d articles)", batchNum, totalBatches, len(batch))

		batchScored, err := g.evaluateBatch(ctx, batch, interests, criteria)
		if err != nil {
			log.Printf("[FeedGenerator] Batch %d failed: %v, using defaults", batchNum, err)
			// On error, use default scores
			for _, a := range batch {
				scored = append(scored, ScoredArticle{Article: a, Score: 0.5})
			}
		} else {
			scored = append(scored, batchScored...)
		}

		// Wait between batches
		if end < len(articles) {
			log.Printf("[FeedGenerator] Waiting %v before next batch...", delayBetweenBatches)
			time.Sleep(delayBetweenBatches)
		}
	}

	return scored, nil
}

// evaluateBatch evaluates a single batch of articles
func (g *FeedGenerator) evaluateBatch(ctx context.Context, articles []Article, interests, criteria string) ([]ScoredArticle, error) {
	// Build prompt
	var articleList strings.Builder
	for i, a := range articles {
		articleList.WriteString(fmt.Sprintf("%d. %s | %s\n", i+1, a.Title, cleanFeedURL(a.URL)))
		if a.Snippet != "" {
			articleList.WriteString(fmt.Sprintf("   %s\n", a.Snippet))
		}
	}

	if criteria == "" {
		criteria = "Focus on new, technical content. Avoid comparisons."
	}

	prompt := fmt.Sprintf(prompts.URLBatchEvaluation, interests, criteria, articleList.String())

	// Call LLM
	resp, err := g.aiProvider.GenerateCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	cleanResp := strings.TrimSpace(resp)
	cleanResp = strings.TrimPrefix(cleanResp, "```json")
	cleanResp = strings.TrimPrefix(cleanResp, "```")
	cleanResp = strings.TrimSuffix(cleanResp, "```")
	cleanResp = strings.TrimSpace(cleanResp)

	var scores []struct {
		URL   string  `json:"url"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(cleanResp), &scores); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Map scores back to articles
	scoreMap := make(map[string]float64)
	for _, s := range scores {
		scoreMap[cleanFeedURL(s.URL)] = s.Score
	}

	scored := make([]ScoredArticle, len(articles))
	for i, a := range articles {
		score, ok := scoreMap[cleanFeedURL(a.URL)]
		if !ok {
			score = 0.5
		}
		scored[i] = ScoredArticle{Article: a, Score: score}
	}

	return scored, nil
}

// cleanFeedURL removes query parameters
func cleanFeedURL(u string) string {
	if idx := strings.Index(u, "?"); idx != -1 {
		return u[:idx]
	}
	return u
}

// truncateFeed limits string length
func truncateFeed(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
