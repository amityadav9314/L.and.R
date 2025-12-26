package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/amityadav/landr/internal/config"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/server"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/token"
	"github.com/amityadav/landr/pkg/pb/auth"
	"github.com/amityadav/landr/pkg/pb/feed"
	"github.com/amityadav/landr/pkg/pb/learning"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	ctx := context.Background()
	st, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer st.Close()

	// Initialize all services
	services := server.Initialize(cfg, st)

	// Setup gRPC server
	tm := token.NewManager(cfg.JWTSecret)
	authInterceptor := middleware.NewAuthInterceptor(tm)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)

	// Register gRPC services
	auth.RegisterAuthServiceServer(grpcServer, services.AuthService)
	learning.RegisterLearningServiceServer(grpcServer, services.LearningService)
	if services.FeedService != nil {
		feed.RegisterFeedServiceServer(grpcServer, services.FeedService)
	}

	reflection.Register(grpcServer)

	// Setup HTTP server (gRPC-Web + REST)
	wrappedServer := server.CreateGRPCWebWrapper(grpcServer)
	httpHandler := server.CreateHTTPHandler(wrappedServer)
	restHandler := server.CreateRESTHandler(services, cfg)
	combinedHandler := server.CreateCombinedHandler(httpHandler, restHandler)
	recoveryHandler := server.CreateRecoveryHandler(combinedHandler)

	// Start HTTP server
	go func() {
		log.Printf("gRPC-Web + REST API listening on :8080")
		if err := http.ListenAndServe(":8080", recoveryHandler); err != nil {
			log.Fatalf("failed to serve grpc-web: %v", err)
		}
	}()

	// Start gRPC server
	log.Printf("Server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
