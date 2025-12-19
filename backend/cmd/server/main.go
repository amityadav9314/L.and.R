package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/scraper"
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
	var aiProvider ai.Provider
	if groqAPIKey != "" && cerebrasAPIKey != "" {
		// Both keys available - use multi-provider (parallel race)
		log.Printf("Using multi-provider AI: Groq + Cerebras (parallel)")
		groq := ai.NewGroqProvider(groqAPIKey)
		cerebras := ai.NewCerebrasProvider(cerebrasAPIKey)
		aiProvider = ai.NewMultiProvider(groq, cerebras)
	} else if groqAPIKey != "" {
		log.Printf("Using single provider: Groq")
		aiProvider = ai.NewGroqProvider(groqAPIKey)
	} else if cerebrasAPIKey != "" {
		log.Printf("Using single provider: Cerebras")
		aiProvider = ai.NewCerebrasProvider(cerebrasAPIKey)
	} else {
		log.Fatal("No AI provider configured. Set GROQ_API_KEY or CEREBRAS_API_KEY")
	}
	learningCore := core.NewLearningCore(st, scr, aiProvider)
	learningSvc := service.NewLearningService(learningCore)

	// Feed Service (requires Tavily API key)
	var feedSvc *service.FeedService
	var feedCore *core.FeedCore
	if tavilyAPIKey != "" {
		log.Printf("Daily Feed feature enabled (Tavily API key found)")
		tavilyClient := tavily.NewClient(tavilyAPIKey)
		feedCore = core.NewFeedCore(st, tavilyClient, aiProvider)
		feedSvc = service.NewFeedService(feedCore)
	} else {
		log.Printf("Daily Feed feature disabled (no TAVILY_API_KEY)")
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

			// Trigger feed generation
			if err := feedCore.GenerateDailyFeedForUser(r.Context(), userID); err != nil {
				log.Printf("[REST] Feed generation failed for %s: %v", email, err)
				http.Error(w, `{"error": "feed generation failed"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "Daily feed refreshed successfully"}`))
			return
		}

		http.NotFound(w, r)
	})

	// Combined handler: REST API routes or gRPC-Web
	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/refresh" {
			restHandler.ServeHTTP(w, r)
			return
		}
		httpHandler.ServeHTTP(w, r)
	})

	log.Printf("Server listening on :50051")
	// Run gRPC-Web on separate port
	go func() {
		log.Printf("gRPC-Web + REST API listening on :8080")
		if err := http.ListenAndServe(":8080", combinedHandler); err != nil {
			log.Fatalf("failed to serve grpc-web: %v", err)
		}
	}()

	// Run standard gRPC on 50051
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
