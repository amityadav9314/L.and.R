package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/amityadav/landr/internal/payment"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/pb/payment_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/amityadav/landr/internal/middleware"
)

// PaymentService implements the gRPC service for payments
type PaymentService struct {
	payment_pb.UnimplementedPaymentServiceServer
	payment *payment.Service
	store   *store.PostgresStore
	keyID   string
}

// NewPaymentService creates a new payment service
func NewPaymentService(p *payment.Service, s *store.PostgresStore, keyID string) *PaymentService {
	return &PaymentService{
		payment: p,
		store:   s,
		keyID:   keyID,
	}
}

// CreateSubscriptionOrder creates a Razorpay order for subscription
func (s *PaymentService) CreateSubscriptionOrder(ctx context.Context, req *payment_pb.CreateSubscriptionOrderRequest) (*payment_pb.CreateSubscriptionOrderResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("[PaymentService] Creating order for user: %s, plan: %s", userID, req.PlanId)

	// In a real app, we'd look up Plan ID to get amount.
	// For "The Scholar" (Pro), amount is ₹199
	amount := 199.0
	currency := "INR"

	notes := map[string]interface{}{
		"user_id": userID,
		"plan":    req.PlanId,
	}

	// Check Payment Flow (popup vs redirect)
	flow := os.Getenv("RAZORPAY_PAYMENT_FLOW") // "redirect" or "popup"

	var paymentLink string
	var orderID string

	if flow == "redirect" {
		// Generate Payment Link
		// Create a unique reference ID
		refID := fmt.Sprintf("pay_%s_%d", userID, time.Now().Unix())

		// Ideally fetch user email, but for MVP we skip or pass empty
		customer := map[string]interface{}{}

		callbackURL := "https://landr.aky.net.in/upgrade?status=success" // simplified

		link, err := s.payment.CreatePaymentLink(amount, currency, refID, "L.and.R Pro Upgrade", customer, notes, callbackURL)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create payment link: %v", err)
		}
		paymentLink = link
		log.Printf("[PaymentService] Generated Payment Link: %s", link)
	} else {
		// Standard Order (Popup)
		oid, err := s.payment.CreateOrder(amount, currency, userID, notes)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
		}
		orderID = oid
	}

	return &payment_pb.CreateSubscriptionOrderResponse{
		OrderId:     orderID,
		Amount:      float32(amount),
		Currency:    currency,
		KeyId:       s.keyID,
		PaymentLink: paymentLink,
	}, nil
}

// VerifyWebhookSignature verifies the webhook signature
func (s *PaymentService) VerifyWebhookSignature(body []byte, signature, webhookSecret string) error {
	return s.payment.VerifyWebhookSignature(body, signature, webhookSecret)
}

// HandleSubscriptionActivated updates user subscription status
func (s *PaymentService) HandleSubscriptionActivated(ctx context.Context, userID, plan, status, subscriptionID string) error {
	// Map Razorpay plan to our internal plan strings
	// In the real world, we'd map plan_id to store.PlanPro etc.
	// For now we assume if we get this callback, it's for PRO.

	sub := &store.Subscription{
		UserID:                 userID,
		Plan:                   store.PlanPro,
		Status:                 store.SubscriptionStatus(status),
		RazorpaySubscriptionID: subscriptionID,
	}
	err := s.store.UpsertSubscription(ctx, sub)
	if err != nil {
		log.Printf("[PaymentService] Failed to upsert subscription: %v", err)
		return err
	}
	log.Printf("[PaymentService] Subscription activated for user %s: %s", userID, subscriptionID)
	return nil
}
