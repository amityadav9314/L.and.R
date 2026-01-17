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

// GenerateFeed generates search-based feed for a user
func (g *FeedGenerator) GenerateFeed(ctx context.Context, userID, userEmail string) error {
	log.Printf("[FeedGenerator] Starting for user: %s (%s)", userEmail, userID)

	// 1. Check Subscription
	sub, err := g.store.GetSubscription(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// 2. Route based on Plan
	if sub.Plan == store.PlanPro {
		return g.GeneratePersonalizedFeed(ctx, userID)
	}

	// Free users get the Global Feed which is generated separately.
	// But if this is called manually, we could perhaps generate a fallback or do nothing.
	log.Printf("[FeedGenerator] User %s is Free tier. Skipping personalized generation.", userID)
	return nil
}

// GenerateGlobalFeed generates a generic tech/learning feed for all free users
func (g *FeedGenerator) GenerateGlobalFeed(ctx context.Context) error {
	log.Printf("[FeedGenerator] Starting Global Feed Generation")

	// 1. Generic Tech/Learning Interests
	interests := []string{
		"latest technology trends 2024",
		"best coding practices and software architecture",
		"new programming languages and frameworks",
		"artificial intelligence developments",
		"productivity tips for developers",
	}

	// 2. Search
	articles, err := g.searchArticles(ctx, interests)
	if err != nil {
		return fmt.Errorf("global search failed: %w", err)
	}
	log.Printf("[FeedGenerator] Global: Found %d articles", len(articles))

	// 3. Evaluate (Generic criteria)
	criteria := "Focus on high-quality, educational technical content. Avoid clickbait."
	scoredArticles, err := g.evaluateArticles(ctx, articles, "Technology, Coding, AI", criteria)
	if err != nil {
		log.Printf("[FeedGenerator] Global evaluation failed: %v", err)
		return nil
	}

	// 4. Store with NULL userID (indicating Global)
	// We need a way to represent Global. Using a fixed UUID or NULL.
	// Let's use a special UUID constant or Handle it in Store.
	// For now, let's use a well-known UUID: "00000000-0000-0000-0000-000000000000" (Nil UUID)
	globalID := "00000000-0000-0000-0000-000000000000"

	stored := 0
	today := time.Now().Truncate(24 * time.Hour)

	for _, sa := range scoredArticles {
		exists, _ := g.store.ArticleURLExists(ctx, globalID, sa.URL)
		if exists {
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
		// We need to ensure the DB constraint allows this.
		// If users table has FK, we need a 'System' user.
		// BETTER: Update schema to allow NULL user_id?
		// OR: Just create a "System User" in migration?
		// Let's assume we use a System User that exists.
		if err := g.store.StoreDailyArticle(ctx, globalID, article); err != nil {
			log.Printf("[FeedGenerator] Failed to store global article: %v", err)
			continue
		}
		stored++
	}

	log.Printf("[FeedGenerator] Stored %d Global articles", stored)
	return nil
}

// GeneratePersonalizedFeed generates feed for a specific PRO user
func (g *FeedGenerator) GeneratePersonalizedFeed(ctx context.Context, userID string) error {
	// 1. Get user preferences
	prefs, err := g.store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get preferences: %w", err)
	}

	if !prefs.FeedEnabled {
		return nil
	}

	interests := strings.Split(prefs.InterestPrompt, ",")
	log.Printf("[FeedGenerator] Generating Personalized for %s, interests: %v", userID, interests)

	// 2. Search
	articles, err := g.searchArticles(ctx, interests)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// 3. Evaluate
	scoredArticles, err := g.evaluateArticles(ctx, articles, prefs.InterestPrompt, prefs.FeedEvalPrompt)
	if err != nil {
		log.Printf("[FeedGenerator] Evaluation failed, using default scores")
		// Fallback
		return nil
	}

	// 4. Store
	stored := 0
	today := time.Now().Truncate(24 * time.Hour)

	for _, sa := range scoredArticles {
		exists, _ := g.store.ArticleURLExists(ctx, userID, sa.URL)
		if exists {
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
			log.Printf("[FeedGenerator] Failed to store article: %v", err)
			continue
		}
		stored++
	}
	log.Printf("[FeedGenerator] Stored %d articles for User %s", stored, userID)
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
