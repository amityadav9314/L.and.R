package firebase

import (
	"context"
	"fmt"
	"log"

	fcm "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Sender handles sending push notifications via Firebase Cloud Messaging
type Sender struct {
	client *messaging.Client
}

// NewSender creates a new Firebase Sender from service account JSON file
func NewSender(serviceAccountPath string) (*Sender, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := fcm.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messaging client: %w", err)
	}

	log.Println("[Firebase] Initialized FCM sender")
	return &Sender{client: client}, nil
}

// NotificationData contains the data for a push notification
type NotificationData struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

// SendNotification sends a push notification to a single device
func (s *Sender) SendNotification(ctx context.Context, data NotificationData) error {
	message := &messaging.Message{
		Token: data.Token,
		Notification: &messaging.Notification{
			Title: data.Title,
			Body:  data.Body,
		},
		Data: data.Data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Icon:  "ic_launcher",
				Color: "#6366F1", // Primary color
			},
		},
	}

	response, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	log.Printf("[Firebase] Notification sent successfully: %s", response)
	return nil
}

// SendToMultiple sends a notification to multiple devices
func (s *Sender) SendToMultiple(ctx context.Context, tokens []string, title, body string, data map[string]string) (int, int) {
	if len(tokens) == 0 {
		return 0, 0
	}

	message := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}

	response, err := s.client.SendEachForMulticast(ctx, message)
	if err != nil {
		log.Printf("[Firebase] Error sending multicast: %v", err)
		return 0, len(tokens)
	}

	log.Printf("[Firebase] Multicast result: %d success, %d failure", response.SuccessCount, response.FailureCount)
	return response.SuccessCount, response.FailureCount
}
