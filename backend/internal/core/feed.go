package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/amityadav/landr/internal/adk/feedagent"
	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/search"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/pb/feed"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FeedCore handles the business logic for the Daily Feed feature
type FeedCore struct {
	store          *store.PostgresStore
	searchRegistry *search.Registry
	scraper        *scraper.Scraper
	aiProvider     ai.Provider
	groqAPIKey     string
}

// NewFeedCore creates a new FeedCore instance
func NewFeedCore(st *store.PostgresStore, searchRegistry *search.Registry, scraper *scraper.Scraper, aiProvider ai.Provider, groqAPIKey string) *FeedCore {
	return &FeedCore{
		store:          st,
		searchRegistry: searchRegistry,
		scraper:        scraper,
		aiProvider:     aiProvider,
		groqAPIKey:     groqAPIKey,
	}
}

// GetFeedPreferences fetches the user's feed preferences
func (c *FeedCore) GetFeedPreferences(ctx context.Context, userID string) (*feed.FeedPreferencesResponse, error) {
	prefs, err := c.store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &feed.FeedPreferencesResponse{
		InterestPrompt: prefs.InterestPrompt,
		FeedEnabled:    prefs.FeedEnabled,
		FeedEvalPrompt: prefs.FeedEvalPrompt, // V2 added
	}, nil
}

// UpdateFeedPreferences updates the user's feed preferences
func (c *FeedCore) UpdateFeedPreferences(ctx context.Context, userID, interestPrompt, evalPrompt string, feedEnabled bool) error {
	return c.store.UpdateFeedPreferences(ctx, userID, interestPrompt, evalPrompt, feedEnabled)
}

// GetDailyFeed fetches articles for a specific date
func (c *FeedCore) GetDailyFeed(ctx context.Context, userID, dateStr string) (*feed.GetDailyFeedResponse, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	articles, err := c.store.GetDailyArticles(ctx, userID, date)
	if err != nil {
		return nil, err
	}

	pbArticles := make([]*feed.Article, len(articles))
	for i, a := range articles {
		isAdded := false
		if matID, err := c.store.GetMaterialBySourceURL(ctx, userID, a.URL); err == nil && matID != "" {
			isAdded = true
		}

		pbArticles[i] = &feed.Article{
			Id:             a.ID,
			Title:          a.Title,
			Url:            a.URL,
			Snippet:        a.Snippet,
			RelevanceScore: float32(a.RelevanceScore),
			CreatedAt:      timestamppb.New(a.CreatedAt),
			Provider:       a.Provider,
			IsAdded:        isAdded,
		}
	}

	return &feed.GetDailyFeedResponse{
		Date:     dateStr,
		Articles: pbArticles,
	}, nil
}

// GetFeedCalendarStatus fetches dates with articles for the calendar view
func (c *FeedCore) GetFeedCalendarStatus(ctx context.Context, userID, monthStr string) (*feed.GetFeedCalendarStatusResponse, error) {
	// Parse "YYYY-MM" format
	t, err := time.Parse("2006-01", monthStr)
	if err != nil {
		// Default to current month if invalid
		t = time.Now()
	}

	days, err := c.store.GetFeedCalendarStatus(ctx, userID, t.Year(), int(t.Month()))
	if err != nil {
		return nil, err
	}

	pbDays := make([]*feed.CalendarDay, len(days))
	for i, d := range days {
		pbDays[i] = &feed.CalendarDay{
			Date:         d.Date.Format("2006-01-02"),
			ArticleCount: d.ArticleCount,
		}
	}

	return &feed.GetFeedCalendarStatusResponse{
		Days: pbDays,
	}, nil
}

// GenerateDailyFeedForUser fetches articles for a single user based on their interest prompt.
// It checks if articles already exist for today - if so, it skips calling Tavily (cached).
// Features:
// - LLM-optimized search query
// - Recency filtering (news topic, last 3 days)
// - Duplicate URL detection (skips already-seen articles)
func (c *FeedCore) GenerateDailyFeedForUser(ctx context.Context, userID string) error {
	log.Printf("[FeedCore.GenerateDailyFeedForUser] Starting for userID: %s", userID)

	// 1. Get user's interest prompt (Quick check before starting heavy workflow)
	prefs, err := c.store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get feed preferences: %w", err)
	}
	if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] User %s has feed disabled or no prompt", userID)
		return nil
	}

	log.Printf("[FeedCore.GenerateDailyFeedForUser] User interests: %s", prefs.InterestPrompt)

	// Check existing articles
	today := time.Now().Truncate(24 * time.Hour)
	existingArticles, err := c.store.GetDailyArticles(ctx, userID, today)
	if err != nil {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] Error checking existing articles: %v", err)
	}

	// Minimum articles before skipping regeneration
	const minDailyArticles = 10
	if len(existingArticles) >= minDailyArticles {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] Already have %d articles today, skipping generation", len(existingArticles))
		return nil
	}

	// 3. Run ADK Agent (V2 Workflow)
	log.Printf("[FeedCore.GenerateDailyFeedForUser] Starting Feed V2 Agent for user %s...", userID)

	deps := feedagent.Dependencies{
		Store:           c.store,
		SearchProviders: c.searchRegistry.GetAll(),
		Scraper:         c.scraper,
		AIProvider:      c.aiProvider,
		GroqAPIKey:      c.groqAPIKey,
	}

	result, err := feedagent.Run(ctx, deps, userID)
	if err != nil {
		return fmt.Errorf("feed agent failed: %w", err)
	}

	log.Printf("[FeedCore.GenerateDailyFeedForUser] Agent completed: %s", result.Summary)
	return nil
}

// GenerateDailyFeedForAllUsers runs the feed generation for all enabled users
// Processes users sequentially with rate limiting to avoid overwhelming external APIs
func (c *FeedCore) GenerateDailyFeedForAllUsers(ctx context.Context) error {
	log.Printf("[FeedCore] Starting daily feed generation...")

	userIDs, err := c.store.GetUsersWithFeedEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users with feed enabled: %w", err)
	}

	log.Printf("[FeedCore] Processing %d users with feed enabled...", len(userIDs))

	successCount := 0
	for i, userID := range userIDs {
		// Rate limit: 2 minute delay between users as requested
		if i > 0 {
			log.Printf("[FeedCore] Rate limiting: waiting 2 minutes before processing user %d/%d...", i+1, len(userIDs))
			time.Sleep(2 * time.Minute)
		}

		if err := c.GenerateDailyFeedForUser(ctx, userID); err != nil {
			log.Printf("[FeedCore] Error for user %s: %v", userID, err)
			// Continue with other users
		} else {
			successCount++
		}
	}

	log.Printf("[FeedCore] Feed generation completed. Success: %d/%d users", successCount, len(userIDs))
	return nil
}
