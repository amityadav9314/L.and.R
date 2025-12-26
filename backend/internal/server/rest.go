package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/amityadav/landr/internal/config"
	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/notifications"
	"github.com/amityadav/landr/internal/store"
)

// CreateRESTHandler creates REST API endpoints
func CreateRESTHandler(services Services, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

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
