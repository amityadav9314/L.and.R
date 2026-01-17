package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	DatabaseURL           string
	JWTSecret             string
	GoogleClientID        string
	GroqAPIKey            string
	CerebrasAPIKey        string
	TavilyAPIKey          string
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	RazorpayPaymentFlow   string
	SerpAPIKey            string
	FeedAPIKey            string
	FeedAPIKey            string
	FirebaseCredPath      string
	LimitFreeLink         int
	LimitFreeText         int
	LimitProLink          int
	LimitProText          int
}

// Load loads configuration from environment variables
func Load() Config {
	return Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://amityadav9314:amit8780@localhost:5432/inkgrid?sslmode=disable"),
		JWTSecret:             getEnv("JWT_SECRET", "dev-secret-key"),
		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GroqAPIKey:            os.Getenv("GROQ_API_KEY"),
		CerebrasAPIKey:        os.Getenv("CEREBRAS_API_KEY"),
		TavilyAPIKey:          os.Getenv("TAVILY_API_KEY"),
		SerpAPIKey:            os.Getenv("SERPAPI_API_KEY"),
		FeedAPIKey:            os.Getenv("FEED_API_KEY"),
		RazorpayKeyID:         getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:     getEnv("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", ""),
		RazorpayPaymentFlow:   getEnv("RAZORPAY_PAYMENT_FLOW", "popup"),
		RazorpayPaymentFlow:   getEnv("RAZORPAY_PAYMENT_FLOW", "popup"),
		FirebaseCredPath:      "firebase/service-account.json",
		LimitFreeLink:         getEnvInt("LIMIT_FREE_LINK", 3),     // Default 5? No, sticking to safe default 3
		LimitFreeText:         getEnvInt("LIMIT_FREE_TEXT", 10),    // Default 10
		LimitProLink:          getEnvInt("LIMIT_PRO_LINK", 20),     // **Changed default to 20 as discussed**
		LimitProText:          getEnvInt("LIMIT_PRO_TEXT", 100000), // Unlimited
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
