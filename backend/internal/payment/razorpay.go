package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/razorpay/razorpay-go"
)

// Service handles payment operations
type Service struct {
	client *razorpay.Client
	secret string
}

// NewService creates a new payment service
func NewService(keyID, keySecret string) *Service {
	client := razorpay.NewClient(keyID, keySecret)
	return &Service{
		client: client,
		secret: keySecret,
	}
}

// CreateOrder creates a Razorpay order
func (s *Service) CreateOrder(amount float64, currency, receipt string, notes map[string]interface{}) (string, error) {
	// Amount in paise (1 INR = 100 paise)
	amountPaise := amount * 100

	data := map[string]interface{}{
		"amount":   amountPaise,
		"currency": currency,
		"receipt":  receipt,
		"notes":    notes,
	}

	body, err := s.client.Order.Create(data, nil)
	if err != nil {
		log.Printf("[Payment] Failed to create order: %v", err)
		return "", fmt.Errorf("failed to create order: %v", err)
	}

	orderID, ok := body["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response from razorpay")
	}

	return orderID, nil
}

// CreatePaymentLink creates a Razorpay Payment Link
func (s *Service) CreatePaymentLink(amount float64, currency, reference, description string, customer map[string]interface{}, notes map[string]interface{}, callbackURL string) (string, error) {
	amountPaise := amount * 100

	data := map[string]interface{}{
		"amount":          amountPaise,
		"currency":        currency,
		"reference_id":    reference,
		"description":     description,
		"customer":        customer,
		"notes":           notes,
		"callback_url":    callbackURL,
		"callback_method": "get",
	}

	body, err := s.client.PaymentLink.Create(data, nil)
	if err != nil {
		log.Printf("[Payment] Failed to create payment link: %v", err)
		return "", fmt.Errorf("failed to create payment link: %v", err)
	}

	shortURL, ok := body["short_url"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response from razorpay (no short_url)")
	}

	return shortURL, nil
}

// VerifySignature verifies the Razorpay signature
func (s *Service) VerifySignature(orderID, paymentID, signature string) error {
	payload := orderID + "|" + paymentID

	h := hmac.New(sha256.New, []byte(s.secret))
	h.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if expectedSignature != signature {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// VerifyWebhookSignature verifies webhook signature
func (s *Service) VerifyWebhookSignature(body []byte, signature, webhookSecret string) error {
	h := hmac.New(sha256.New, []byte(webhookSecret))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	if expectedSignature != signature {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}
