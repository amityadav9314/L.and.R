package fx

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/amityadav/landr/internal/config"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/notifications"
	"github.com/amityadav/landr/internal/server"
	"github.com/amityadav/landr/internal/service"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/token"
	"github.com/amityadav/landr/pkg/pb/auth"
	"github.com/amityadav/landr/pkg/pb/feed"
	"github.com/amityadav/landr/pkg/pb/learning"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ServerModule provides gRPC and HTTP servers
var ServerModule = fx.Module("server",
	fx.Provide(NewGRPCServer),
	fx.Invoke(
		RegisterGRPCServices,
		StartServers,
		StartNotificationWorker,
	),
)

// NewGRPCServer creates configured gRPC server with auth interceptor
func NewGRPCServer(tm *token.Manager) *grpc.Server {
	authInterceptor := middleware.NewAuthInterceptor(tm)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.Unary()),
	)
	reflection.Register(srv)
	log.Printf("[FX] gRPC Server created")
	return srv
}

// GRPCServicesParams groups all gRPC services for registration
type GRPCServicesParams struct {
	fx.In
	Server          *grpc.Server
	AuthService     *service.AuthService
	LearningService *service.LearningService
	FeedService     *service.FeedService `optional:"true"` // Optional
}

// RegisterGRPCServices registers all gRPC services with the server
func RegisterGRPCServices(p GRPCServicesParams) {
	auth.RegisterAuthServiceServer(p.Server, p.AuthService)
	learning.RegisterLearningServiceServer(p.Server, p.LearningService)

	if p.FeedService != nil {
		feed.RegisterFeedServiceServer(p.Server, p.FeedService)
		log.Printf("[FX] Registered: AuthService, LearningService, FeedService")
	} else {
		log.Printf("[FX] Registered: AuthService, LearningService (FeedService disabled)")
	}
}

// ServerParams groups dependencies for starting servers
type ServerParams struct {
	fx.In
	Lifecycle       fx.Lifecycle
	GRPCServer      *grpc.Server
	Store           *store.PostgresStore
	AuthService     *service.AuthService
	LearningService *service.LearningService
	FeedService     *service.FeedService  `optional:"true"`
	FeedCore        *core.FeedCore        `optional:"true"`
	NotifWorker     *notifications.Worker `optional:"true"`
	TokenManager    *token.Manager
	Config          config.Config
}

// StartServers starts gRPC and HTTP servers with lifecycle management
func StartServers(p ServerParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start gRPC server
			lis, err := net.Listen("tcp", ":50051")
			if err != nil {
				return err
			}

			go func() {
				log.Printf("[FX] gRPC Server listening on :50051")
				if err := p.GRPCServer.Serve(lis); err != nil {
					log.Printf("[FX] gRPC Server error: %v", err)
				}
			}()

			// Start HTTP server (gRPC-Web + REST)
			wrappedServer := server.CreateGRPCWebWrapper(p.GRPCServer)
			httpHandler := server.CreateHTTPHandler(wrappedServer)

			// Create REST handler using server.Services
			serverServices := server.Services{
				Store:           p.Store,
				AuthService:     p.AuthService,
				LearningService: p.LearningService,
				FeedService:     p.FeedService,
				FeedCore:        p.FeedCore,
				NotifWorker:     p.NotifWorker,
				TokenManager:    p.TokenManager,
			}
			restHandler := server.CreateRESTHandler(serverServices, p.Config)
			combinedHandler := server.CreateCombinedHandler(httpHandler, restHandler)
			recoveryHandler := server.CreateRecoveryHandler(combinedHandler)

			go func() {
				log.Printf("[FX] HTTP Server (gRPC-Web + REST) listening on :8080")
				if err := http.ListenAndServe(":8080", recoveryHandler); err != nil {
					log.Printf("[FX] HTTP Server error: %v", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Printf("[FX] Shutting down servers...")
			p.GRPCServer.GracefulStop()
			return nil
		},
	})
}

// WorkerStartParams for optional worker injection
type WorkerStartParams struct {
	fx.In
	Lifecycle fx.Lifecycle
	Worker    *notifications.Worker `optional:"true"`
}

// StartNotificationWorker starts the notification worker if available
func StartNotificationWorker(p WorkerStartParams) {
	if p.Worker == nil {
		return
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Worker.Start()
			log.Printf("[FX] NotificationWorker started (Feed: 6 AM, Notifications: 9 AM IST)")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Worker.Stop()
			return nil
		},
	})
}
