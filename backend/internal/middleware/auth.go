package middleware

import (
	"context"
	"strings"

	"github.com/amityadav/landr/internal/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthInterceptor is a gRPC interceptor that extracts and verifies JWT tokens
type AuthInterceptor struct {
	tokenManager *token.Manager
	// Methods that don't require authentication
	publicMethods map[string]bool
}

func NewAuthInterceptor(tm *token.Manager) *AuthInterceptor {
	return &AuthInterceptor{
		tokenManager: tm,
		publicMethods: map[string]bool{
			"/auth.AuthService/Login": true, // Login doesn't require auth
		},
	}
}

// Unary returns a server interceptor function to authenticate and authorize unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if this method requires authentication
		if interceptor.publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		// Token format: "Bearer <token>"
		accessToken := values[0]
		if !strings.HasPrefix(accessToken, "Bearer ") {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		accessToken = strings.TrimPrefix(accessToken, "Bearer ")

		// Verify token and extract user ID
		userID, err := interceptor.tokenManager.Verify(accessToken)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add user ID to context
		ctx = context.WithValue(ctx, UserIDKey, userID)

		return handler(ctx, req)
	}
}

// GetUserID extracts the user ID from context
func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return "", status.Errorf(codes.Internal, "user ID not found in context")
	}
	return userID, nil
}

