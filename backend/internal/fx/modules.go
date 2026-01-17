package fx

import (
	"context"
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

	"github.com/amityadav/landr/internal/payment"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/internal/token"
	"go.uber.org/fx"
)

// ============================================================================
// FX MODULES - Group related providers together (like Spring @Configuration)
// ============================================================================

// ConfigModule provides application configuration
var ConfigModule = fx.Module("config",
	fx.Provide(config.Load),
)

// StoreModule provides database connectivity
var StoreModule = fx.Module("store",
	fx.Provide(NewPostgresStore),
)

// TokenModule provides JWT token management
var TokenModule = fx.Module("token",
	fx.Provide(NewTokenManager),
)

// ScraperModule provides web scraping capabilities
var ScraperModule = fx.Module("scraper",
	fx.Provide(scraper.NewScraper),
)

// AIModule provides AI/LLM providers
var AIModule = fx.Module("ai",
	fx.Provide(
		NewLearningAIProvider,
		NewFeedAIProvider,
	),
)

// SearchModule provides search registry with all search providers
var SearchModule = fx.Module("search",
	fx.Provide(NewSearchRegistry),
)

// CoreModule provides business logic cores
var CoreModule = fx.Module("core",
	fx.Provide(
		NewAuthCore,
		NewLearningCore,
		NewFeedCore,
	),
)

// ServiceModule provides gRPC service implementations
var ServiceModule = fx.Module("service",
	fx.Provide(
		service.NewAuthService,
		NewLearningService,
		NewFeedService,
		NewPaymentService,
	),
)

// NotificationModule provides notification worker
var NotificationModule = fx.Module("notification",
	fx.Provide(
		NewFirebaseSender,
		NewNotificationWorker,
	),
)

// PaymentModule provides payment service
var PaymentModule = fx.Module("payment",
	fx.Provide(NewRazorpayService),
)

// ============================================================================
// PROVIDER FUNCTIONS - Constructors that FX will call automatically
// ============================================================================

// NewPostgresStore creates database connection
func NewPostgresStore(cfg config.Config) (*store.PostgresStore, error) {
	ctx := context.Background()
	st, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	log.Printf("[FX] PostgresStore initialized")
	return st, nil
}

// NewTokenManager creates JWT token manager
func NewTokenManager(cfg config.Config) *token.Manager {
	tm := token.NewManager(cfg.JWTSecret)
	log.Printf("[FX] TokenManager initialized")
	return tm
}

// LearningAIProvider is a named type for the learning AI provider
type LearningAIProvider struct {
	fx.Out
	Provider ai.Provider `name:"learning"`
}

// FeedAIProvider is a named type for the feed AI provider
type FeedAIProvider struct {
	fx.Out
	Provider ai.Provider `name:"feed"`
}

// NewLearningAIProvider creates AI provider for flashcard generation
func NewLearningAIProvider(cfg config.Config) LearningAIProvider {
	var provider ai.Provider

	if cfg.GroqAPIKey != "" {
		groq := ai.NewLLMProvider("groq", cfg.GroqAPIKey, models.TaskFlashcardModel)
		if cfg.CerebrasAPIKey != "" {
			cerebras := ai.NewLLMProvider("cerebras", cfg.CerebrasAPIKey, models.TaskFlashcardModel)
			provider = ai.NewMultiProvider(groq, cerebras)
			log.Printf("[FX] LearningAIProvider initialized (MultiProvider: Groq + Cerebras)")
		} else {
			provider = groq
			log.Printf("[FX] LearningAIProvider initialized (Groq)")
		}
	} else if cfg.CerebrasAPIKey != "" {
		provider = ai.NewLLMProvider("cerebras", cfg.CerebrasAPIKey, models.TaskFlashcardModel)
		log.Printf("[FX] LearningAIProvider initialized (Cerebras)")
	} else {
		log.Fatal("[FX] No AI provider configured. Set GROQ_API_KEY or CEREBRAS_API_KEY")
	}

	return LearningAIProvider{Provider: provider}
}

// NewFeedAIProvider creates AI provider for feed/agent tasks
func NewFeedAIProvider(cfg config.Config) FeedAIProvider {
	var provider ai.Provider

	if cfg.GroqAPIKey != "" {
		provider = ai.NewLLMProvider("groq", cfg.GroqAPIKey, models.TaskAgentDailyFeedModel)
		log.Printf("[FX] FeedAIProvider initialized (Groq)")
	} else if cfg.CerebrasAPIKey != "" {
		provider = ai.NewLLMProvider("cerebras", cfg.CerebrasAPIKey, models.TaskAgentDailyFeedModel)
		log.Printf("[FX] FeedAIProvider initialized (Cerebras)")
	} else {
		log.Fatal("[FX] No AI provider configured for feed")
	}

	return FeedAIProvider{Provider: provider}
}

// NewSearchRegistry creates search registry with all available providers
func NewSearchRegistry(cfg config.Config) *search.Registry {
	registry := search.NewRegistry()

	if cfg.TavilyAPIKey != "" {
		registry.Register(tavily.NewClient(cfg.TavilyAPIKey))
		log.Printf("[FX] SearchRegistry: Tavily registered")
	}

	if cfg.SerpAPIKey != "" {
		registry.Register(serpapi.NewClient(cfg.SerpAPIKey))
		log.Printf("[FX] SearchRegistry: SerpApi registered")
	}

	log.Printf("[FX] SearchRegistry initialized with %d providers", registry.Count())
	return registry
}

// NewAuthCore creates auth business logic
func NewAuthCore(st *store.PostgresStore, tm *token.Manager, cfg config.Config) *core.AuthCore {
	c := core.NewAuthCore(st, tm, cfg.GoogleClientID)
	log.Printf("[FX] AuthCore initialized")
	return c
}

// LearningCoreParams groups dependencies for LearningCore (fx.In for named deps)
type LearningCoreParams struct {
	fx.In
	Store            *store.PostgresStore
	Scraper          *scraper.Scraper
	LearningProvider ai.Provider `name:"learning"`
}

// NewLearningCore creates learning business logic
func NewLearningCore(p LearningCoreParams) *core.LearningCore {
	c := core.NewLearningCore(p.Store, p.Scraper, p.LearningProvider)
	log.Printf("[FX] LearningCore initialized")
	return c
}

// FeedCoreParams groups dependencies for FeedCore
type FeedCoreParams struct {
	fx.In
	Store          *store.PostgresStore
	SearchRegistry *search.Registry
	Scraper        *scraper.Scraper
	FeedProvider   ai.Provider `name:"feed"`
	Config         config.Config
}

// NewFeedCore creates feed business logic (optional - returns nil if no search providers)
func NewFeedCore(p FeedCoreParams) *core.FeedCore {
	if p.SearchRegistry.Count() == 0 {
		log.Printf("[FX] FeedCore disabled (no search providers configured)")
		return nil
	}

	c := core.NewFeedCore(p.Store, p.SearchRegistry, p.Scraper, p.FeedProvider, p.Config.GroqAPIKey, p.Config.CerebrasAPIKey)
	log.Printf("[FX] FeedCore initialized")
	return c
}

// NewLearningService creates learning gRPC service
func NewLearningService(c *core.LearningCore, st *store.PostgresStore) *service.LearningService {
	svc := service.NewLearningService(c, st)
	log.Printf("[FX] LearningService initialized")
	return svc
}

// NewFeedService creates feed gRPC service (optional)
func NewFeedService(c *core.FeedCore, st *store.PostgresStore) *service.FeedService {
	if c == nil {
		log.Printf("[FX] FeedService disabled (no FeedCore)")
		return nil
	}
	svc := service.NewFeedService(c, st)
	log.Printf("[FX] FeedService initialized")
	return svc
}

// NewPaymentService creates payment gRPC service (optional)
func NewPaymentService(p *payment.Service, st *store.PostgresStore, cfg config.Config) *service.PaymentService {
	if p == nil {
		log.Printf("[FX] PaymentService disabled (no Payment provider)")
		return nil
	}
	svc := service.NewPaymentService(p, st, cfg.RazorpayKeyID, cfg.RazorpayPaymentFlow)
	log.Printf("[FX] PaymentService initialized (Flow: %s)", cfg.RazorpayPaymentFlow)
	return svc
}

// NewFirebaseSender creates Firebase Cloud Messaging sender (optional)
func NewFirebaseSender(cfg config.Config) *firebase.Sender {
	if _, err := os.Stat(cfg.FirebaseCredPath); err != nil {
		log.Printf("[FX] FirebaseSender disabled (no %s)", cfg.FirebaseCredPath)
		return nil
	}

	sender, err := firebase.NewSender(cfg.FirebaseCredPath)
	if err != nil {
		log.Printf("[FX] FirebaseSender failed: %v", err)
		return nil
	}

	log.Printf("[FX] FirebaseSender initialized")
	return sender
}

// NotificationWorkerParams groups dependencies for notification worker
type NotificationWorkerParams struct {
	fx.In
	Store        *store.PostgresStore
	LearningCore *core.LearningCore
	FeedCore     *core.FeedCore `optional:"true"` // Optional dependency
	FCM          *firebase.Sender
}

// NewNotificationWorker creates notification worker (optional)
func NewNotificationWorker(p NotificationWorkerParams) *notifications.Worker {
	if p.FCM == nil {
		log.Printf("[FX] NotificationWorker disabled (no Firebase)")
		return nil
	}

	worker := notifications.NewWorker(p.Store, p.LearningCore, p.FCM)
	if p.FeedCore != nil {
		worker.SetFeedCore(p.FeedCore)
	}

	log.Printf("[FX] NotificationWorker initialized")
	return worker
}

// NewRazorpayService creates Razorpay service
func NewRazorpayService(cfg config.Config) *payment.Service {
	if cfg.RazorpayKeyID == "" {
		log.Printf("[FX] PaymentService disabled (no Razorpay key)")
		return nil
	}
	svc := payment.NewService(cfg.RazorpayKeyID, cfg.RazorpayKeySecret)
	log.Printf("[FX] PaymentService initialized")
	return svc
}
