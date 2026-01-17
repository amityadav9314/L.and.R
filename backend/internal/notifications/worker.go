package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/firebase"
	"github.com/amityadav/landr/internal/store"
	"github.com/robfig/cron/v3"
)

const APP_NAME = "L.and.R"

// Worker handles scheduled notification tasks
type Worker struct {
	store        *store.PostgresStore
	learningCore *core.LearningCore
	feedCore     *core.FeedCore // Optional: for daily feed generation
	fcm          *firebase.Sender
	cron         *cron.Cron
}

// NewWorker creates a new notification worker
func NewWorker(store *store.PostgresStore, learningCore *core.LearningCore, fcm *firebase.Sender) *Worker {
	return &Worker{
		store:        store,
		learningCore: learningCore,
		fcm:          fcm,
		cron:         cron.New(cron.WithLocation(time.FixedZone("IST", 5*60*60+30*60))), // IST timezone
	}
}

// SetFeedCore adds the FeedCore for daily article generation
func (w *Worker) SetFeedCore(feedCore *core.FeedCore) {
	w.feedCore = feedCore
}

// Start starts the notification worker with daily schedule at 9 AM IST
func (w *Worker) Start() {
	log.Println("[Worker] Starting daily schedulers...")

	// Schedule feed generation at 6 AM IST (before notifications)
	if w.feedCore != nil {
		_, err := w.cron.AddFunc("0 6 * * *", func() {
			// Run async to not block the scheduler
			go func() {
				log.Println("[Worker] Running daily feed generation job (async)...")
				ctx := context.Background()
				if err := w.feedCore.GenerateDailyFeedForAllUsers(ctx); err != nil {
					log.Printf("[Worker] Feed generation failed: %v", err)
				}
			}()
		})

		// Also schedule Global Feed generation (once daily)
		// We can run this separately or as part of the same block. Let's make it explicit.
		_, _ = w.cron.AddFunc("0 6 * * *", func() {
			go func() {
				log.Println("[Worker] Running Global Feed generation (async)...")
				ctx := context.Background()
				if err := w.feedCore.GenerateGlobalFeed(ctx); err != nil {
					log.Printf("[Worker] Global Feed generation failed: %v", err)
				}
			}()
		})

		if err != nil {
			log.Printf("[Worker] Failed to schedule feed job: %v", err)
		} else {
			log.Println("[Worker] Scheduled daily feed generation (Global + Personal) at 6:00 AM IST")
		}
	}

	// Schedule notifications at 9 AM IST daily
	_, err := w.cron.AddFunc("0 9 * * *", func() {
		// Run async to not block the scheduler
		go func() {
			log.Println("[Worker] Running daily notification job (async)...")
			w.SendDailyNotifications()
		}()
	})
	if err != nil {
		log.Printf("[Worker] Failed to schedule notification job: %v", err)
		return
	}

	w.cron.Start()
	log.Println("[Worker] Scheduled daily notifications at 9:00 AM IST")
}

// Stop stops the notification worker
func (w *Worker) Stop() {
	w.cron.Stop()
	log.Println("[NotificationWorker] Stopped")
}

// SendDailyNotifications sends personalized notifications to all users with due materials
// Processes users sequentially with rate limiting to avoid overwhelming the system
func (w *Worker) SendDailyNotifications() {
	ctx := context.Background()

	// Get all users with device tokens
	userIDs, err := w.store.GetAllUsersWithTokens(ctx)
	if err != nil {
		log.Printf("[Worker] Failed to get users: %v", err)
		return
	}

	log.Printf("[Worker] Checking %d users for due materials...", len(userIDs))

	sentCount := 0
	for i, userID := range userIDs {
		// Rate limit: 2 minute delay between users as requested
		if i > 0 {
			log.Printf("[Worker] Rate limiting: waiting 2 minutes before processing user %d/%d...", i+1, len(userIDs))
			time.Sleep(2 * time.Minute)
		}

		// Check if user has due materials
		_, hasDue, materialCount, firstTitle, err := w.learningCore.GetNotificationStatus(ctx, userID)
		if err != nil {
			log.Printf("[Worker] Error checking user %s: %v", userID, err)
			continue
		}

		if !hasDue || materialCount == 0 {
			continue // No due materials for this user
		}

		// Get user's device tokens
		tokens, err := w.store.GetDeviceTokens(ctx, userID)
		if err != nil || len(tokens) == 0 {
			continue
		}

		// Build notification content
		title := fmt.Sprintf("%s - Review Due! 📚", APP_NAME)
		body := w.buildNotificationBody(firstTitle, materialCount)

		// Send to all user's devices
		success, _ := w.fcm.SendToMultiple(ctx, tokens, title, body, map[string]string{
			"type":  "due_materials",
			"count": fmt.Sprintf("%d", materialCount),
		})

		if success > 0 {
			sentCount++
		}
	}

	log.Printf("[Worker] Daily notifications complete. Sent to %d users.", sentCount)
}

// buildNotificationBody creates the notification body text
func (w *Worker) buildNotificationBody(firstTitle string, count int32) string {
	if firstTitle == "" {
		firstTitle = "Untitled"
	}

	if count == 1 {
		return fmt.Sprintf("\"%s\" is due for revision.", firstTitle)
	}

	othersCount := count - 1
	if othersCount == 1 {
		return fmt.Sprintf("\"%s\" and 1 other are due for revision.", firstTitle)
	}

	return fmt.Sprintf("\"%s\" and %d others are due for revision.", firstTitle, othersCount)
}

// SendTestNotification sends a test notification to a specific user (for debugging)
func (w *Worker) SendTestNotification(ctx context.Context, userID string) error {
	tokens, err := w.store.GetDeviceTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get tokens: %w", err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("no device tokens found for user")
	}

	title := fmt.Sprintf("%s - Test Notification 🧪", APP_NAME)
	body := "This is a test notification from your backend!"

	success, failed := w.fcm.SendToMultiple(ctx, tokens, title, body, nil)
	log.Printf("[NotificationWorker] Test notification: %d success, %d failed", success, failed)

	if success == 0 {
		return fmt.Errorf("failed to send to any device")
	}
	return nil
}
