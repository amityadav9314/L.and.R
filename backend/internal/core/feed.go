package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/pkg/pb/feed"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FeedCore handles the business logic for the Daily Feed feature
type FeedCore struct {
	store        *store.PostgresStore
	tavilyClient *tavily.Client
	aiProvider   ai.Provider
}

// NewFeedCore creates a new FeedCore instance
func NewFeedCore(st *store.PostgresStore, tavilyClient *tavily.Client, aiProvider ai.Provider) *FeedCore {
	return &FeedCore{
		store:        st,
		tavilyClient: tavilyClient,
		aiProvider:   aiProvider,
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
	}, nil
}

// UpdateFeedPreferences updates the user's feed preferences
func (c *FeedCore) UpdateFeedPreferences(ctx context.Context, userID, interestPrompt string, feedEnabled bool) error {
	return c.store.UpdateFeedPreferences(ctx, userID, interestPrompt, feedEnabled)
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
		pbArticles[i] = &feed.Article{
			Id:             a.ID,
			Title:          a.Title,
			Url:            a.URL,
			Snippet:        a.Snippet,
			RelevanceScore: float32(a.RelevanceScore),
			CreatedAt:      timestamppb.New(a.CreatedAt),
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

	// 1. Get user's interest prompt
	prefs, err := c.store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get feed preferences: %w", err)
	}
	if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] User %s has feed disabled or no prompt", userID)
		return nil
	}

	// 2. Check if articles already exist for today (caching)
	today := time.Now().Truncate(24 * time.Hour)
	existingArticles, err := c.store.GetDailyArticles(ctx, userID, today)
	if err != nil {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] Error checking existing articles: %v", err)
		// Continue anyway
	}

	if len(existingArticles) > 0 {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] Articles already exist for today (%d found), skipping Tavily call", len(existingArticles))
		return nil // Cache hit - don't call Tavily
	}

	// 3. Use LLM to optimize the search query
	log.Printf("[FeedCore.GenerateDailyFeedForUser] Optimizing search query with LLM...")
	optimizedQuery, err := c.aiProvider.OptimizeSearchQuery(prefs.InterestPrompt)
	if err != nil {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] Query optimization failed, using original: %v", err)
		optimizedQuery = prefs.InterestPrompt
	}
	log.Printf("[FeedCore.GenerateDailyFeedForUser] Original: %q -> Optimized: %q", prefs.InterestPrompt, optimizedQuery)

	// 4. Call Tavily with recency filtering (news topic, last 3 days)
	log.Printf("[FeedCore.GenerateDailyFeedForUser] Calling Tavily API with news filter...")
	searchResp, err := c.tavilyClient.SearchWithOptions(optimizedQuery, tavily.SearchOptions{
		MaxResults: 15, // Fetch more since we'll filter duplicates
		NewsOnly:   true,
		Days:       3, // Only articles from last 3 days
	})
	if err != nil {
		return fmt.Errorf("tavily search failed: %w", err)
	}

	if len(searchResp.Results) == 0 {
		log.Printf("[FeedCore.GenerateDailyFeedForUser] No search results for user %s", userID)
		return nil
	}

	// 5. Store articles, skipping duplicates (URLs already in DB)
	storedCount := 0
	skippedCount := 0
	for _, result := range searchResp.Results {
		// Check if URL already exists for this user (prevents duplicates across days)
		exists, err := c.store.ArticleURLExists(ctx, userID, result.URL)
		if err != nil {
			log.Printf("[FeedCore.GenerateDailyFeedForUser] Error checking URL: %v", err)
			// Continue anyway
		}
		if exists {
			skippedCount++
			continue // Skip duplicate
		}

		article := &store.DailyArticle{
			Title:          result.Title,
			URL:            result.URL,
			Snippet:        result.Content,
			RelevanceScore: result.Score,
			SuggestedDate:  today,
		}
		if err := c.store.StoreDailyArticle(ctx, userID, article); err != nil {
			log.Printf("[FeedCore.GenerateDailyFeedForUser] Failed to store article: %v", err)
			// Continue with other articles
		} else {
			storedCount++
		}
	}

	log.Printf("[FeedCore.GenerateDailyFeedForUser] Stored %d articles (skipped %d duplicates) for user %s", storedCount, skippedCount, userID)
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
		// Rate limit: 2 second delay between users (to respect Tavily API limits)
		if i > 0 {
			time.Sleep(2 * time.Second)
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
