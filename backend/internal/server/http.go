package server

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

// CreateGRPCWebWrapper creates a gRPC-Web wrapper with CORS
func CreateGRPCWebWrapper(grpcServer *grpc.Server) *grpcweb.WrappedGrpcServer {
	return grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true // Allow all origins for development
		}),
		grpcweb.WithAllowedRequestHeaders([]string{
			"x-grpc-web", "content-type", "x-user-agent", "grpc-timeout",
			"authorization", "x-requested-with", "cache-control", "range",
		}),
	)
}

// CreateHTTPHandler creates the main HTTP handler with CORS
func CreateHTTPHandler(wrappedServer *grpcweb.WrappedGrpcServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization,x-grpc-web,x-user-agent,grpc-timeout")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length,Content-Range,grpc-status,grpc-message,grpc-status-details-bin")

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Max-Age", "1728000")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		wrappedServer.ServeHTTP(w, r)
	}
}

// CreateCombinedHandler combines REST and gRPC-Web handlers
func CreateCombinedHandler(httpHandler, restHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/feed/refresh" ||
			r.URL.Path == "/api/notification/test" ||
			r.URL.Path == "/api/notification/daily" ||
			r.URL.Path == "/api/privacy-policy" {
			restHandler.ServeHTTP(w, r)
			return
		}
		httpHandler.ServeHTTP(w, r)
	}
}

// CreateRecoveryHandler wraps handler with panic recovery
func CreateRecoveryHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", err, debug.Stack())
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
			}
		}()
		handler.ServeHTTP(w, r)
	}
}
