package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/amityadav/landr/internal/config"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/notifications"
	"github.com/amityadav/landr/internal/service"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/token"
)

// Services groups all service dependencies for REST handlers
type Services struct {
	Store           *store.PostgresStore
	AuthService     *service.AuthService
	LearningService *service.LearningService
	FeedService     *service.FeedService
	FeedCore        *core.FeedCore
	NotifWorker     *notifications.Worker
	TokenManager    *token.Manager
}

// CreateRESTHandler creates REST API endpoints
func CreateRESTHandler(services Services, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.URL.Path {
		case "/api/feed/refresh":
			handleFeedRefresh(w, r, services.Store, services.FeedCore, cfg.FeedAPIKey)
		case "/api/notification/test":
			handleNotificationTest(w, r, services.Store, services.NotifWorker, cfg.FeedAPIKey)
		case "/api/notification/daily":
			handleNotificationDaily(w, r, services.NotifWorker, cfg.FeedAPIKey)
		case "/api/privacy-policy":
			handlePrivacyPolicy(w, r)
		case "/api/admin/users":
			handleGetAllUsers(w, r, services.Store, services.TokenManager)
		case "/api/admin/set-admin":
			handleSetAdminStatus(w, r, services.Store, cfg.FeedAPIKey)
		default:
			http.NotFound(w, r)
		}
	}
}

func handleFeedRefresh(w http.ResponseWriter, r *http.Request, st *store.PostgresStore, feedCore *core.FeedCore, feedAPIKey string) {
	if feedAPIKey == "" {
		http.Error(w, `{"error": "FEED_API_KEY not configured on server"}`, http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("X-API-Key") != feedAPIKey {
		http.Error(w, `{"error": "unauthorized - invalid or missing X-API-Key header"}`, http.StatusUnauthorized)
		return
	}

	if feedCore == nil {
		http.Error(w, `{"error": "Daily Feed feature is disabled"}`, http.StatusServiceUnavailable)
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

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		log.Printf("[REST] Starting background feed generation for user %s", userID)
		if err := feedCore.GenerateDailyFeedForUser(bgCtx, userID); err != nil {
			log.Printf("[REST] Feed generation failed for %s: %v", email, err)
		} else {
			log.Printf("[REST] Feed generation completed for %s", email)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "accepted", "message": "Daily feed refresh started in background"}`))
}

func handleNotificationTest(w http.ResponseWriter, r *http.Request, st *store.PostgresStore, notifWorker *notifications.Worker, feedAPIKey string) {
	if feedAPIKey == "" || r.Header.Get("X-API-Key") != feedAPIKey {
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
}

func handleNotificationDaily(w http.ResponseWriter, r *http.Request, notifWorker *notifications.Worker, feedAPIKey string) {
	if feedAPIKey == "" || r.Header.Get("X-API-Key") != feedAPIKey {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if notifWorker == nil {
		http.Error(w, `{"error": "Push notifications not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	log.Println("[REST] Manually triggering daily notification job...")
	go notifWorker.SendDailyNotifications()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Daily notification job triggered"}`))
}

func handlePrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Privacy Policy - L.and.R</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; color: #333; }
        h1 { color: #1a73e8; }
        h2 { color: #444; margin-top: 30px; }
        .updated { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <h1>Privacy Policy</h1>
    <p class="updated">Last updated: January 2, 2026</p>
    
    <h2>Introduction</h2>
    <p>L.and.R ("we", "our", or "us") is committed to protecting your privacy. This Privacy Policy explains how we collect, use, and safeguard your information when you use our mobile application.</p>
    
    <h2>Information We Collect</h2>
    <p><strong>Account Information:</strong> When you sign in with Google, we receive your email address and profile name to create your account.</p>
    <p><strong>Learning Materials:</strong> Content you add to the app (URLs, notes, flashcards) is stored securely to provide the learning service.</p>
    <p><strong>Usage Data:</strong> We collect anonymous usage statistics to improve the app experience.</p>
    
    <h2>How We Use Your Information</h2>
    <ul>
        <li>To provide and maintain our learning service</li>
        <li>To send you revision reminders and notifications (with your permission)</li>
        <li>To generate personalized daily feed content based on your preferences</li>
        <li>To improve our app and user experience</li>
    </ul>
    
    <h2>Data Storage and Security</h2>
    <p>Your data is stored securely on our servers. We implement industry-standard security measures to protect your information.</p>
    
    <h2>Third-Party Services</h2>
    <p>We use the following third-party services:</p>
    <ul>
        <li><strong>Google Sign-In:</strong> For authentication</li>
        <li><strong>Firebase Cloud Messaging:</strong> For push notifications</li>
    </ul>
    
    <h2>Your Rights</h2>
    <p>You can request deletion of your account and associated data at any time by contacting us.</p>
    
    <h2>Children's Privacy</h2>
    <p>Our app is not intended for children under 13. We do not knowingly collect information from children under 13.</p>
    
    <h2>Contact Us</h2>
    <p>If you have questions about this Privacy Policy, please contact us at: <a href="mailto:amityadav9314@gmail.com">amityadav9314@gmail.com</a></p>
</body>
</html>`

	w.Write([]byte(html))
}

func handleGetAllUsers(w http.ResponseWriter, r *http.Request, st *store.PostgresStore, tm *token.Manager) {
	if r.Method != "GET" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Verify the user is authenticated and is an admin
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error": "unauthorized - missing Authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Extract token from "Bearer <token>"
	tokenStr := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenStr = authHeader[7:]
	}

	// Validate token and get user ID
	userID, err := tm.Verify(tokenStr)
	if err != nil {
		log.Printf("[REST] handleGetAllUsers - token validation failed: %v", err)
		http.Error(w, `{"error": "unauthorized - invalid token"}`, http.StatusUnauthorized)
		return
	}

	// Get user from store to check admin status
	user, err := st.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Printf("[REST] handleGetAllUsers - failed to get user: %v", err)
		http.Error(w, `{"error": "unauthorized - user not found"}`, http.StatusUnauthorized)
		return
	}

	if !user.IsAdmin {
		log.Printf("[REST] handleGetAllUsers - non-admin user attempted access: %s", user.Email)
		http.Error(w, `{"error": "forbidden - admin access required"}`, http.StatusForbidden)
		return
	}

	// Parse pagination params
	page := 1
	pageSize := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	users, totalCount, err := st.GetAllUsersForAdmin(r.Context(), page, pageSize)
	if err != nil {
		log.Printf("[REST] handleGetAllUsers - failed to get users: %v", err)
		http.Error(w, `{"error": "failed to get users"}`, http.StatusInternalServerError)
		return
	}

	// Build JSON response manually
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonUsers := "["
	for i, u := range users {
		if i > 0 {
			jsonUsers += ","
		}
		jsonUsers += `{"id":"` + u.ID + `","email":"` + u.Email + `","name":"` + u.Name + `","picture":"` + u.Picture + `","is_admin":` + boolToString(u.IsAdmin) + `,"created_at":"` + u.CreatedAt.Format("2006-01-02T15:04:05Z07:00") + `"}`
	}
	jsonUsers += "]"

	totalPages := (totalCount + pageSize - 1) / pageSize
	response := fmt.Sprintf(`{"users":%s,"page":%d,"page_size":%d,"total_count":%d,"total_pages":%d}`, jsonUsers, page, pageSize, totalCount, totalPages)
	w.Write([]byte(response))
}

func handleSetAdminStatus(w http.ResponseWriter, r *http.Request, st *store.PostgresStore, feedAPIKey string) {
	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if feedAPIKey == "" || r.Header.Get("X-API-Key") != feedAPIKey {
		http.Error(w, `{"error": "unauthorized - invalid or missing X-API-Key header"}`, http.StatusUnauthorized)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, `{"error": "email query parameter is required"}`, http.StatusBadRequest)
		return
	}

	isAdminStr := r.URL.Query().Get("is_admin")
	if isAdminStr == "" {
		http.Error(w, `{"error": "is_admin query parameter is required"}`, http.StatusBadRequest)
		return
	}

	isAdmin := isAdminStr == "true" || isAdminStr == "1"

	if err := st.SetUserAdminStatus(r.Context(), email, isAdmin); err != nil {
		log.Printf("[REST] handleSetAdminStatus - failed: %v", err)
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("[REST] handleSetAdminStatus - set admin=%v for email: %s", isAdmin, email)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success", "message": "Admin status updated", "email": "` + email + `", "is_admin": ` + boolToString(isAdmin) + `}`))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
