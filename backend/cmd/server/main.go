package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/ai/models"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/firebase"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/notifications"
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/service"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/internal/token"
	"github.com/amityadav/landr/pkg/pb/auth"
	"github.com/amityadav/landr/pkg/pb/feed"
	"github.com/amityadav/landr/pkg/pb/learning"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// 1. Configuration
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://amityadav9314:amit8780@localhost:5432/inkgrid?sslmode=disable"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key"
	}
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	cerebrasAPIKey := os.Getenv("CEREBRAS_API_KEY")
	tavilyAPIKey := os.Getenv("TAVILY_API_KEY")
	serpapiAPIKey := os.Getenv("SERPAPI_API_KEY")
	feedAPIKey := os.Getenv("FEED_API_KEY")

	// 2. Database
	ctx := context.Background()
	st, err := store.NewPostgresStore(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer st.Close()

	// 3. Services
	tm := token.NewManager(jwtSecret)

	// Auth
	authCore := core.NewAuthCore(st, tm, googleClientID)
	authSvc := service.NewAuthService(authCore)

	// Learning - Multi-provider AI (Groq + Cerebras racing in parallel)
	scr := scraper.NewScraper()
	// Learning - Multi-provider AI (Groq + Cerebras racing in parallel)
	var learningProvider ai.Provider
	var feedProvider ai.Provider // Distinct provider for Feed Agent

	// Helper to create safe provider (fallback logic could be added here)
	createProvider := func(name, key, model string) ai.Provider {
		return ai.NewLLMProvider(name, key, model)
	}

	if groqAPIKey != "" {
		log.Printf("Initializing AI Providers (Primary: Groq)")

		// 1. Learning Provider (Flashcards) - uses TaskFlashcardModel (GPT-OSS)
		// If we have Cerebras too, we can still use MultiProvider for race, or just simple Groq.
		// For simplicity/robustness, let's stick to Groq for Learning unless Multi is desired.
		// Detailed requirement: "Feed uses Qwen, Flashcards uses GPT-OSS"

		groqFlashcard := createProvider("groq", groqAPIKey, models.TaskFlashcardModel)

		if cerebrasAPIKey != "" {
			// Multi-provider race for Flashcards (Speed)
			cerebrasFlashcard := createProvider("cerebras", cerebrasAPIKey, models.TaskFlashcardModel)
			learningProvider = ai.NewMultiProvider(groqFlashcard, cerebrasFlashcard)
		} else {
			learningProvider = groqFlashcard
		}

		// 2. Feed Provider (Agent) - uses TaskAgentDailyFeedModel (Qwen/Llama)
		// Agent needs robust model, avoiding MultiProvider race race complexity for stateful agent.
		feedProvider = createProvider("groq", groqAPIKey, models.TaskAgentDailyFeedModel)

	} else if cerebrasAPIKey != "" {
		log.Printf("Initializing AI Providers (Primary: Cerebras)")
		// Fallback to Cerebras for everything
		learningProvider = createProvider("cerebras", cerebrasAPIKey, models.TaskFlashcardModel)
		feedProvider = createProvider("cerebras", cerebrasAPIKey, models.TaskAgentDailyFeedModel)
	} else {
		log.Fatal("No AI provider configured. Set GROQ_API_KEY or CEREBRAS_API_KEY")
	}

	learningCore := core.NewLearningCore(st, scr, learningProvider)
	learningSvc := service.NewLearningService(learningCore, st)

	// Feed Service (requires Tavily or SerpApi API key)
	var feedSvc *service.FeedService
	var feedCore *core.FeedCore
	if tavilyAPIKey != "" || serpapiAPIKey != "" {
		log.Printf("Daily Feed feature enabled")
		var tavilyClient *tavily.Client
		if tavilyAPIKey != "" {
			log.Printf("  - Tavily search enabled")
			tavilyClient = tavily.NewClient(tavilyAPIKey)
		}
		var serpapiClient *serpapi.Client
		if serpapiAPIKey != "" {
			log.Printf("  - SerpApi (Google) search enabled")
			serpapiClient = serpapi.NewClient(serpapiAPIKey)
		}

		feedCore = core.NewFeedCore(st, tavilyClient, serpapiClient, scr, feedProvider, groqAPIKey)
		feedSvc = service.NewFeedService(feedCore)
	} else {
		log.Printf("Daily Feed feature disabled (no TAVILY_API_KEY or SERPAPI_API_KEY)")
	}

	// Firebase Push Notifications (optional)
	var notifWorker *notifications.Worker
	firebaseServiceAccountPath := "firebase/service-account.json"
	if _, err := os.Stat(firebaseServiceAccountPath); err == nil {
		fcmSender, err := firebase.NewSender(firebaseServiceAccountPath)
		if err != nil {
			log.Printf("WARNING: Failed to initialize Firebase: %v", err)
		} else {
			notifWorker = notifications.NewWorker(st, learningCore, fcmSender)
			// Add feedCore for daily article generation (6 AM IST)
			if feedCore != nil {
				notifWorker.SetFeedCore(feedCore)
			}
			notifWorker.Start()
			log.Printf("Worker started (Feed: 6 AM, Notifications: 9 AM IST)")
		}
	} else {
		log.Printf("Push notifications disabled (no firebase/service-account.json)")
	}

	// 4. Auth Interceptor
	authInterceptor := middleware.NewAuthInterceptor(tm)

	// 5. gRPC Server with Auth Interceptor
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)
	auth.RegisterAuthServiceServer(s, authSvc)
	learning.RegisterLearningServiceServer(s, learningSvc)
	if feedSvc != nil {
		feed.RegisterFeedServiceServer(s, feedSvc)
	}

	// Enable reflection for debugging (e.g. with grpcurl)
	reflection.Register(s)

	// 6. gRPC-Web Wrapper with CORS
	wrappedServer := grpcweb.WrapServer(s,
		grpcweb.WithOriginFunc(func(origin string) bool {
			// Allow all origins for development
			// In production, you should restrict this to specific domains
			return true
		}),
		grpcweb.WithAllowedRequestHeaders([]string{
			"x-grpc-web",
			"content-type",
			"x-user-agent",
			"grpc-timeout",
			"authorization",
			"x-requested-with",
			"cache-control",
			"range",
		}),
	)

	// Custom HTTP handler with CORS
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,x-grpc-web,x-user-agent,grpc-timeout")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Range,grpc-status,grpc-message,grpc-status-details-bin")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Max-Age", "1728000")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass to gRPC-Web handler
		wrappedServer.ServeHTTP(w, r)
	})

	// Create REST API handler for manual feed refresh
	restHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.URL.Path == "/api/feed/refresh" && r.Method == "POST" {
			// API Key authentication
			authHeader := r.Header.Get("X-API-Key")
			if feedAPIKey == "" {
				http.Error(w, `{"error": "FEED_API_KEY not configured on server"}`, http.StatusServiceUnavailable)
				return
			}
			if authHeader != feedAPIKey {
				http.Error(w, `{"error": "unauthorized - invalid or missing X-API-Key header"}`, http.StatusUnauthorized)
				return
			}

			if feedCore == nil {
				http.Error(w, `{"error": "Daily Feed feature is disabled. Set TAVILY_API_KEY."}`, http.StatusServiceUnavailable)
				return
			}

			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, `{"error": "email query parameter is required"}`, http.StatusBadRequest)
				return
			}

			// Look up user by email
			userID, err := st.GetUserByEmail(r.Context(), email)
			if err != nil {
				http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
				return
			}

			// Trigger feed generation in background (async)
			// Use context.Background() to prevent cancellation if client disconnects
			go func() {
				// Use a long timeout context (e.g. 30 mins) just in case
				bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()

				log.Printf("[REST] Starting background feed generation for user %s", userID)
				if err := feedCore.GenerateDailyFeedForUser(bgCtx, userID); err != nil {
					log.Printf("[REST] Background feed generation failed for %s: %v", email, err)
				} else {
					log.Printf("[REST] Background feed generation completed for %s", email)
				}
			}()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"status": "accepted", "message": "Daily feed refresh started in background"}`))
			return
		}

		// Test notification endpoint
		if r.URL.Path == "/api/notification/test" && r.Method == "POST" {
			authHeader := r.Header.Get("X-API-Key")
			if feedAPIKey == "" || authHeader != feedAPIKey {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if notifWorker == nil {
				http.Error(w, `{"error": "Push notifications not enabled"}`, http.StatusServiceUnavailable)
				return
			}

			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, `{"error": "email query parameter is required"}`, http.StatusBadRequest)
				return
			}

			userID, err := st.GetUserByEmail(r.Context(), email)
			if err != nil {
				http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
				return
			}

			if err := notifWorker.SendTestNotification(r.Context(), userID); err != nil {
				log.Printf("[REST] Test notification failed: %v", err)
				http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Test notification sent"}`))
			return
		}

		// Trigger daily notification job manually (checks due materials for all users)
		if r.URL.Path == "/api/notification/daily" && r.Method == "POST" {
			authHeader := r.Header.Get("X-API-Key")
			if feedAPIKey == "" || authHeader != feedAPIKey {
				http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if notifWorker == nil {
				http.Error(w, `{"error": "Push notifications not enabled"}`, http.StatusServiceUnavailable)
				return
			}

			log.Println("[REST] Manually triggering daily notification job...")
			go notifWorker.SendDailyNotifications() // Run async

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Daily notification job triggered"}`))
			return
		}

		http.NotFound(w, r)
	})

	// Combined handler: REST API routes or gRPC-Web
	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/refresh" || r.URL.Path == "/api/notification/test" || r.URL.Path == "/api/notification/daily" {
			restHandler.ServeHTTP(w, r)
			return
		}
		httpHandler.ServeHTTP(w, r)
	})

	// Recovery middleware to prevent panics from crashing the server
	recoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", err, debug.Stack())
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			}
		}()
		combinedHandler.ServeHTTP(w, r)
	})

	log.Printf("Server listening on :50051")
	// Run gRPC-Web on separate port
	go func() {
		log.Printf("gRPC-Web + REST API listening on :8080")
		if err := http.ListenAndServe(":8080", recoveryHandler); err != nil {
			log.Fatalf("failed to serve grpc-web: %v", err)
		}
	}()

	// Run standard gRPC on 50051
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
