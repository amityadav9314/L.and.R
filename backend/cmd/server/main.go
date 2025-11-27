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
	"github.com/amityadav/landr/internal/token"
	"github.com/amityadav/landr/pkg/pb/auth"
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

	// Learning
	scr := scraper.NewScraper()
	aiClient := ai.NewClient(groqAPIKey)
	learningCore := core.NewLearningCore(st, scr, aiClient)
	learningSvc := service.NewLearningService(learningCore)

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

	// Enable reflection for debugging (e.g. with grpcurl)
	reflection.Register(s)

	// 6. gRPC-Web Wrapper
	wrappedServer := grpcweb.WrapServer(s,
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
		grpcweb.WithAllowedRequestHeaders([]string{"*"}),
	)

	log.Printf("Server listening on :50051")
	// Run gRPC-Web on separate port
	go func() {
		log.Printf("gRPC-Web listening on :8080")
		if err := http.ListenAndServe(":8080", wrappedServer); err != nil {
			log.Fatalf("failed to serve grpc-web: %v", err)
		}
	}()

	// Run standard gRPC on 50051
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
