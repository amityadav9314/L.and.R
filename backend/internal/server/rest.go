package server

import (
	"context"
	"encoding/json"
	"io"
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
	PaymentService  *service.PaymentService
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
		case "/api/payment/webhook":
			handlePaymentWebhook(w, r, services.PaymentService, cfg.RazorpayWebhookSecret)
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

	// Build response using proper JSON marshaling
	type userResponse struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		IsAdmin       bool   `json:"is_admin"`
		CreatedAt     string `json:"created_at"`
		MaterialCount int    `json:"material_count"`
	}

	type paginatedResponse struct {
		Users      []userResponse `json:"users"`
		Page       int            `json:"page"`
		PageSize   int            `json:"page_size"`
		TotalCount int            `json:"total_count"`
		TotalPages int            `json:"total_pages"`
	}

	userList := make([]userResponse, 0, len(users))
	for _, u := range users {
		userList = append(userList, userResponse{
			ID:            u.ID,
			Email:         u.Email,
			Name:          u.Name,
			Picture:       u.Picture,
			IsAdmin:       u.IsAdmin,
			CreatedAt:     u.CreatedAt.Format(time.RFC3339),
			MaterialCount: u.MaterialCount,
		})
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	response := paginatedResponse{
		Users:      userList,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request, paymentService *service.PaymentService, webhookSecret string) {
	if r.Method != "POST" {
		http.Error(w, `{"error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if paymentService == nil {
		log.Printf("[REST] Payment service not enabled, ignoring webhook")
		w.WriteHeader(http.StatusOK) // Return 200 to acknowledge but ignore
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[REST] Failed to read webhook body: %v", err)
		http.Error(w, `{"error": "failed to read body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature
	signature := r.Header.Get("X-Razorpay-Signature")
	if signature == "" {
		http.Error(w, `{"error": "missing signature"}`, http.StatusBadRequest)
		return
	}

	if err := paymentService.VerifyWebhookSignature(body, signature, webhookSecret); err != nil {
		log.Printf("[REST] Webhook signature verification failed: %v", err)
		http.Error(w, `{"error": "invalid signature"}`, http.StatusUnauthorized)
		return
	}

	// Parse JSON
	var payload struct {
		Event   string `json:"event"`
		Payload struct {
			Subscription struct {
				Entity struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Notes  struct {
						UserID string `json:"user_id"`
					} `json:"notes"`
				} `json:"entity"`
			} `json:"subscription"`
			Payment struct {
				Entity struct {
					ID    string `json:"id"`
					Notes struct {
						UserID string `json:"user_id"`
					} `json:"notes"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[REST] Failed to parse webhook JSON: %v", err)
		http.Error(w, `{"error": "invalid json"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[REST] Received webhook event: %s", payload.Event)

	// Handle Subscription Events
	if payload.Event == "subscription.activated" || payload.Event == "subscription.charged" {
		sub := payload.Payload.Subscription.Entity
		userID := sub.Notes.UserID
		if userID != "" {
			if err := paymentService.HandleSubscriptionActivated(r.Context(), userID, "PRO", "active", sub.ID); err != nil {
				log.Printf("[REST] Failed to handle subscription activation: %v", err)
				http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("[REST] Webhook received for subscription but no user_id found. SubID: %s", sub.ID)
		}
	}

	// Handle One-Time Payment Events (Order based)
	if payload.Event == "payment.captured" {
		pay := payload.Payload.Payment.Entity
		userID := pay.Notes.UserID
		// For one-time payment, we treat it as "active" subscription for MVP.
		// We use payment ID as reference if no subscription ID.
		if userID != "" {
			log.Printf("[REST] Payment captured for user: %s (PaymentID: %s)", userID, pay.ID)
			// Logic: Set status to ACTIVE.
			// Ideally we should track "Pro until..." logic. For now, just set to PRO.
			// Using "order_payment" prefix for ID to distinguish.
			fakeSubID := "pay_" + pay.ID

			if err := paymentService.HandleSubscriptionActivated(r.Context(), userID, "PRO", "active", fakeSubID); err != nil {
				log.Printf("[REST] Failed to activate pro from payment: %v", err)
				http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("[REST] Webhook received for payment but no user_id found. PayID: %s", pay.ID)
		}
	}

	w.WriteHeader(http.StatusOK)
}
