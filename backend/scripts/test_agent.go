package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/amityadav/landr/internal/adk/feedagent"
	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load env
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	groqKey := os.Getenv("GROQ_API_KEY")
	tavilyKey := os.Getenv("TAVILY_API_KEY")
	serpKey := os.Getenv("SERPAPI_API_KEY")

	if groqKey == "" || dbURL == "" {
		log.Fatal("DATABASE_URL and GROQ_API_KEY are required")
	}

	// 2. Setup Store
	st, err := store.NewPostgresStore(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/test_agent.go <email_or_user_id>")
	}
	input := os.Args[1]

	// 3. Resolve User ID if email is provided
	userID := input
	if strings.Contains(input, "@") {
		log.Printf("Resolving email %s to UserID...", input)
		uid, err := st.GetUserByEmail(context.Background(), input)
		if err != nil {
			log.Fatalf("Failed to find user with email %s: %v", input, err)
		}
		userID = uid
		log.Printf("Found UserID: %s", userID)
	}

	log.Printf("Starting Agent Test for User: %s", userID)

	deps := feedagent.Dependencies{
		Store:      st,
		Tavily:     tavily.NewClient(tavilyKey),
		SerpApi:    serpapi.NewClient(serpKey),
		GroqAPIKey: groqKey,
	}

	// 4. Run Agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result, err := feedagent.Run(ctx, deps, userID)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	log.Printf("\n--- AGENT SUMMARY ---\n%s\n--------------------", result.Summary)
}
