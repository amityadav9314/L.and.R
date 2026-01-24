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
	FirebaseCredPath      string
	LimitFreeLink         int
	LimitFreeText         int
	LimitProLink          int
	LimitProText          int
	LimitFreeImage        int
	LimitFreeYoutube      int
	LimitProImage         int
	LimitProYoutube       int
	ProAccessDays         int
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
		FirebaseCredPath:      "firebase/service-account.json",
		LimitFreeLink:         getEnvIntOrPanic("LIMIT_FREE_LINK"),
		LimitFreeText:         getEnvIntOrPanic("LIMIT_FREE_TEXT"),
		LimitProLink:          getEnvIntOrPanic("LIMIT_PRO_LINK"),
		LimitProText:          getEnvIntOrPanic("LIMIT_PRO_TEXT"),
		LimitFreeImage:        getEnvInt("LIMIT_FREE_IMAGE", 5),
		LimitFreeYoutube:      getEnvInt("LIMIT_FREE_YOUTUBE", 3),
		LimitProImage:         getEnvInt("LIMIT_PRO_IMAGE", 100),
		LimitProYoutube:       getEnvInt("LIMIT_PRO_YOUTUBE", 50),
		ProAccessDays:         getEnvInt("PRO_ACCESS_DAYS", 30),
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

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Missing required environment variable: " + key)
	}
	return value
}

func getEnvIntOrPanic(key string) int {
	value := os.Getenv(key)
	if value == "" {
		panic("Missing required environment variable: " + key)
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		panic("Invalid integer for environment variable " + key + ": " + value)
	}
	return i
}
