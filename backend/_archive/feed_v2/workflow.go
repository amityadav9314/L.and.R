package feed_v2

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
)

// WorkflowDependencies holds all services needed for the V2 feed workflow
type WorkflowDependencies struct {
	Store *store.PostgresStore
	// TODO we have have n numaber of providers for search, should we keep on adding them here???
	Tavily  *tavily.Client
	SerpApi *serpapi.Client

	Scraper   *scraper.Scraper
	AI        ai.Provider
	GroqModel string // e.g. "llama-3.3-70b-versatile"
}

// Config controls the workflow limits
type Config struct {
	MaxArticlesPerDay int
	SearchMaxResults  int // per provider
	MinRelevanceScore float64
	MaxSearchLoops    int
}

var DefaultConfig = Config{
	// TODO - should we not keep this in constants file??
	MaxArticlesPerDay: 10,
	SearchMaxResults:  10,
	MinRelevanceScore: 0.6,
	MaxSearchLoops:    3,
}

// Workflow orchestrates the V2 Daily Feed generation
type Workflow struct {
	deps   WorkflowDependencies
	config Config
}

func NewWorkflow(deps WorkflowDependencies, cfg Config) *Workflow {
	return &Workflow{
		deps:   deps,
		config: cfg,
	}
}

// Run executes the feed generation for a single user
func (w *Workflow) Run(ctx context.Context, userID string) error {
	log.Printf("[FeedV2] Starting workflow for user: %s", userID)

	// 1. Fetch Preferences
	prefs, err := w.deps.Store.GetFeedPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch prefs: %w", err)
	}
	if !prefs.FeedEnabled || prefs.InterestPrompt == "" {
		log.Printf("[FeedV2] Feed disabled or empty prompt. Exiting.")
		return nil
	}

	// 2. Main Loop
	today := time.Now().Truncate(24 * time.Hour)
	storedCount := 0

	// Check existing count first
	existing, err := w.deps.Store.GetDailyArticles(ctx, userID, today)
	if err == nil {
		storedCount = len(existing)
	}

	loops := 0
	for storedCount < w.config.MaxArticlesPerDay && loops < w.config.MaxSearchLoops {
		loops++
		log.Printf("[FeedV2] Loop %d/%d (Stored: %d/%d)", loops, w.config.MaxSearchLoops, storedCount, w.config.MaxArticlesPerDay)

		// A. Generate Query
		query, err := w.generateSearchQuery(ctx, prefs.InterestPrompt)
		if err != nil {
			log.Printf("[FeedV2] Query gen failed: %v", err)
			continue
		}
		log.Printf("[FeedV2] Generated Query: %s", query)

		// B. Search (Parallel Providers)
		candidates := w.searchParameters(query)
		if len(candidates) == 0 {
			log.Printf("[FeedV2] No search results found.")
			continue
		}

		// C. Process Candidates (Scrape -> Summarize -> Eval)
		// We process them in batches or all at once? All at once with concurrency limit.
		// Limit concurrency to polite levels (e.g. 5)
		newArticles := w.processCandidates(ctx, userID, candidates, prefs.InterestPrompt, prefs.FeedEvalPrompt)

		// D. Store
		for _, art := range newArticles {
			if storedCount >= w.config.MaxArticlesPerDay {
				break
			}
			if err := w.deps.Store.StoreDailyArticle(ctx, userID, art); err == nil {
				storedCount++
				log.Printf("[FeedV2] Stored '%s' (Score: %.2f, Provider: %s)", art.Title, art.RelevanceScore, art.Provider)
			}
		}
	}

	log.Printf("[FeedV2] Finished. Total Articles for today: %d", storedCount)
	return nil
}

// CandidateURL represents a search result to be processed
type CandidateURL struct {
	Title    string
	URL      string
	Provider string // "tavily" or "google"
	Snippet  string // Original snippet from search
}

// searchParameters runs searches in parallel
func (w *Workflow) searchParameters(query string) []CandidateURL {
	var candidates []CandidateURL

	// TODO we must have a list of providers somewhere in some contants. Then we must loop over those providers and run all steps below.
	// 1. Tavily
	if w.deps.Tavily != nil {
		log.Printf("[FeedV2] Searching Tavily (Sync)...")
		res, err := w.deps.Tavily.SearchWithOptions(query, tavily.SearchOptions{
			NewsOnly:   true,
			Days:       7,
			MaxResults: w.config.SearchMaxResults,
		})
		if err == nil {
			for _, r := range res.Results {
				candidates = append(candidates, CandidateURL{
					Title:    r.Title,
					URL:      r.URL,
					Provider: "tavily",
					Snippet:  r.Content,
				})
			}
		} else {
			log.Printf("[FeedV2] Tavily search failed: %v", err)
		}
	}

	// 2. SerpApi
	//if w.deps.SerpApi != nil {
	//	log.Printf("[FeedV2] Searching SerpApi (Sync)...")
	//	res, err := w.deps.SerpApi.SearchNews(query)
	//	if err == nil {
	//		for _, r := range res.Results {
	//			candidates = append(candidates, CandidateURL{
	//				Title:    r.Title,
	//				URL:      r.URL,
	//				Provider: "google",
	//				Snippet:  r.Snippet,
	//			})
	//		}
	//	} else {
	//		log.Printf("[FeedV2] SerpApi search failed: %v", err)
	//	}
	//}

	return w.deduplicate(candidates)
}

func (w *Workflow) deduplicate(candidates []CandidateURL) []CandidateURL {
	seen := make(map[string]bool)
	var unique []CandidateURL
	for _, c := range candidates {
		// Simple normalization
		u := strings.TrimRight(c.URL, "/")
		if !seen[u] {
			seen[u] = true
			unique = append(unique, c)
		}
	}
	return unique
}

func (w *Workflow) processCandidates(ctx context.Context, userID string, candidates []CandidateURL, interestPrompt, evalPrompt string) []*store.DailyArticle {
	var results []*store.DailyArticle
	var mu sync.Mutex
	sem := make(chan struct{}, 1) // sequential to avoid 429
	var wg sync.WaitGroup

	for _, c := range candidates {
		// Check DB for duplicate URL before scraping
		exists, _ := w.deps.Store.ArticleURLExists(ctx, userID, c.URL)
		if exists {
			continue
		}

		wg.Add(1)
		go func(cand CandidateURL) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			// 1. Skip Scrape (Optimization)
			// We use the snippet/content provided by the search provider (Tavily)
			// to save on scraper credits and time.
			content := cand.Snippet
			if len(content) < 100 {
				return // snippet too short for meaningful summary
			}

			// 2. Summarize
			summary, err := w.deps.AI.GenerateSummary(content)
			if err != nil {
				return
			}

			// 3. Evaluate
			score, err := w.evaluateArticle(ctx, summary, interestPrompt, evalPrompt)
			if err != nil {
				return
			}

			if score >= w.config.MinRelevanceScore {
				art := &store.DailyArticle{
					Title:          cand.Title,
					URL:            cand.URL,
					Snippet:        summary, // Store generated summary as snippet for UI
					RelevanceScore: score,
					SuggestedDate:  time.Now(),
					Provider:       cand.Provider,
				}
				mu.Lock()
				results = append(results, art)
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()
	return results
}

// AI Helpers (Wrappers around AI Provider)

func (w *Workflow) generateSearchQuery(ctx context.Context, interests string) (string, error) {
	// TODO - we must use package prompts. There is already one written over there, please check and merge and use that
	prompt := fmt.Sprintf(`You are a news curator. The user likes: "%s".
Generate ONE specific, high-quality search query to find recent news articles for this user.
The query MUST be under 350 characters.
Return ONLY the query text. Do not use quotes.`, interests)

	query, err := w.deps.AI.GenerateCompletion(prompt)
	if err != nil {
		return "", err
	}

	finalQuery := strings.TrimSpace(query)
	// Remove surrounding quotes if LLM added them
	finalQuery = strings.Trim(finalQuery, "\"")

	// Enforce Tavily's 400 char limit strictly
	if len(finalQuery) > 380 {
		finalQuery = finalQuery[:380]
	}

	log.Printf("[FeedV2] Generated Search Query: %s", finalQuery)
	return finalQuery, nil
}

func (w *Workflow) evaluateArticle(ctx context.Context, summary, interests, evalPrompt string) (float64, error) {
	criteria := evalPrompt
	if criteria == "" {
		criteria = "Ensure the article is informative, relevant to their interests, and not clickbait."
	}

	// TODO - we must use package prompts. There is already one written over there, please check and merge and use that
	prompt := fmt.Sprintf(`Evaluate this article summary for a user.
User Interests: "%s"
Evaluation Criteria: "%s"

Article Summary:
"%s"

Task: Rate the relevance and quality of this article on a scale of 0.0 to 1.0.
0.0 = Totally irrelevant / Spam
1.0 = Perfect match / Must read

Return ONLY the float score (e.g. 0.85).`, interests, criteria, summary)

	resp, err := w.deps.AI.GenerateCompletion(prompt)
	if err != nil {
		return 0, err
	}

	// Parse float
	var score float64
	_, err = fmt.Sscanf(strings.TrimSpace(resp), "%f", &score)
	if err != nil {
		// Log warning but fallback to safe score?
		// Better to reject if we can't parse?
		// Let's assume 0.0 if parse fails to be safe.
		return 0.0, nil
	}
	return score, nil
}
