package main

import (
	"log"

	appfx "github.com/amityadav/landr/internal/fx"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Create and run the FX application
	// FX automatically:
	// 1. Resolves all dependencies (like Spring @Autowired)
	// 2. Manages lifecycle (OnStart/OnStop hooks)
	// 3. Handles graceful shutdown on SIGINT/SIGTERM
	app := fx.New(
		// Modules group related providers (like Spring @Configuration)
		appfx.ConfigModule,       // Provides: config.Config
		appfx.StoreModule,        // Provides: *store.PostgresStore
		appfx.SettingsModule,     // Provides: *settings.Service (database-backed)
		appfx.TokenModule,        // Provides: *token.Manager
		appfx.ScraperModule,      // Provides: *scraper.Scraper
		appfx.AIModule,           // Provides: ai.Provider (named: "learning", "feed")
		appfx.SearchModule,       // Provides: *search.Registry
		appfx.CoreModule,         // Provides: *core.AuthCore, *core.LearningCore, *core.FeedCore
		appfx.ServiceModule,      // Provides: *service.AuthService, *service.LearningService, *service.FeedService
		appfx.NotificationModule, // Provides: *firebase.Sender, *notifications.Worker
		appfx.PaymentModule,      // Provides: *payment.Service (Razorpay)
		appfx.ServerModule,       // Starts gRPC + HTTP servers, registers services

		// Use simple console logger for cleaner output
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ConsoleLogger{W: log.Writer()}
		}),
	)

	// Run blocks until the app receives a shutdown signal
	app.Run()
}
