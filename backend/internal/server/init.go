package server

import (
	"log"
	"os"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/ai/models"
	"github.com/amityadav/landr/internal/config"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/firebase"
	"github.com/amityadav/landr/internal/notifications"
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/search"
	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/service"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/internal/token"
)

// Services holds all initialized services
type Services struct {
	Store           *store.PostgresStore
	AuthService     *service.AuthService
	LearningService *service.LearningService
	FeedService     *service.FeedService
	FeedCore        *core.FeedCore
	NotifWorker     *notifications.Worker
}

// Initialize sets up all application services
func Initialize(cfg config.Config, st *store.PostgresStore) Services {
	tm := token.NewManager(cfg.JWTSecret)
	scr := scraper.NewScraper()

	// Auth service
	authCore := core.NewAuthCore(st, tm, cfg.GoogleClientID)
	authSvc := service.NewAuthService(authCore)

	// AI providers
	learningProvider, feedProvider := initializeAIProviders(cfg)

	// Learning service
	learningCore := core.NewLearningCore(st, scr, learningProvider)
	learningSvc := service.NewLearningService(learningCore, st)

	// Feed service (optional)
	feedSvc, feedCore := initializeFeedService(cfg, st, scr, feedProvider)

	// Notification worker (optional)
	notifWorker := initializeNotificationWorker(cfg, st, learningCore, feedCore)

	return Services{
		Store:           st,
		AuthService:     authSvc,
		LearningService: learningSvc,
		FeedService:     feedSvc,
		FeedCore:        feedCore,
		NotifWorker:     notifWorker,
	}
}

func initializeAIProviders(cfg config.Config) (learning ai.Provider, feed ai.Provider) {
	createProvider := func(name, key, model string) ai.Provider {
		return ai.NewLLMProvider(name, key, model)
	}

	if cfg.GroqAPIKey != "" {
		log.Printf("Initializing AI Providers (Primary: Groq)")

		groqFlashcard := createProvider("groq", cfg.GroqAPIKey, models.TaskFlashcardModel)
		if cfg.CerebrasAPIKey != "" {
			cerebrasFlashcard := createProvider("cerebras", cfg.CerebrasAPIKey, models.TaskFlashcardModel)
			learning = ai.NewMultiProvider(groqFlashcard, cerebrasFlashcard)
		} else {
			learning = groqFlashcard
		}

		feed = createProvider("groq", cfg.GroqAPIKey, models.TaskAgentDailyFeedModel)

	} else if cfg.CerebrasAPIKey != "" {
		log.Printf("Initializing AI Providers (Primary: Cerebras)")
		learning = createProvider("cerebras", cfg.CerebrasAPIKey, models.TaskFlashcardModel)
		feed = createProvider("cerebras", cfg.CerebrasAPIKey, models.TaskAgentDailyFeedModel)
	} else {
		log.Fatal("No AI provider configured. Set GROQ_API_KEY or CEREBRAS_API_KEY")
	}

	return learning, feed
}

func initializeFeedService(cfg config.Config, st *store.PostgresStore, scr *scraper.Scraper, feedProvider ai.Provider) (*service.FeedService, *core.FeedCore) {
	if cfg.TavilyAPIKey == "" && cfg.SerpAPIKey == "" {
		log.Printf("Daily Feed feature disabled (no TAVILY_API_KEY or SERPAPI_API_KEY)")
		return nil, nil
	}

	log.Printf("Daily Feed feature enabled")

	searchRegistry := search.NewRegistry()

	if cfg.TavilyAPIKey != "" {
		log.Printf("  - Registering Tavily search provider")
		searchRegistry.Register(tavily.NewClient(cfg.TavilyAPIKey))
	}

	if cfg.SerpAPIKey != "" {
		log.Printf("  - Registering SerpApi (Google) search provider")
		searchRegistry.Register(serpapi.NewClient(cfg.SerpAPIKey))
	}

	log.Printf("  - Total search providers registered: %d", searchRegistry.Count())

	feedCore := core.NewFeedCore(st, searchRegistry, scr, feedProvider, cfg.GroqAPIKey)
	feedSvc := service.NewFeedService(feedCore)

	return feedSvc, feedCore
}

func initializeNotificationWorker(cfg config.Config, st *store.PostgresStore, learningCore *core.LearningCore, feedCore *core.FeedCore) *notifications.Worker {
	if _, err := os.Stat(cfg.FirebaseCredPath); err != nil {
		log.Printf("Push notifications disabled (no %s)", cfg.FirebaseCredPath)
		return nil
	}

	fcmSender, err := firebase.NewSender(cfg.FirebaseCredPath)
	if err != nil {
		log.Printf("WARNING: Failed to initialize Firebase: %v", err)
		return nil
	}

	worker := notifications.NewWorker(st, learningCore, fcmSender)
	if feedCore != nil {
		worker.SetFeedCore(feedCore)
	}

	worker.Start()
	log.Printf("Worker started (Feed: 6 AM, Notifications: 9 AM IST)")

	return worker
}
