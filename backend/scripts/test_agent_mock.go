package main

import (
	"context"
	"fmt"
	"iter"
	"log"
	"os"
	"strings"
	"time"

	"github.com/amityadav/landr/internal/adk/tools"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/adk/model/groq"
	"github.com/amityadav/landr/prompts"
	"github.com/joho/godotenv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// MockSearchNewsTool returns dummy articles without calling APIs
func MockSearchNewsTool() tool.Tool {
	return &tools.Simple{
		NameVal: "search_news",
		DescVal: prompts.ToolSearchNewsDesc,
		Fn: func(args map[string]interface{}) (string, error) {
			queries, _ := args["queries"].([]interface{})
			log.Printf("[MOCK] search_news called with %d queries: %v", len(queries), queries)

			// Dummy results
			return `Found 3 articles:

Title: AI Agents are the future of software
URL: https://example.com/ai-agents
Content: A deep dive into how autonomous agents like AutoGPT and specialized ADK agents are changing the landscape of software development.
Source: Tavily
---

Title: New Groq LPU benchmarks released
URL: https://example.com/groq-benchmarks
Snippet: Groq's latest LPU shows 10x performance increase in token generation for Llama 3 models compared to traditional GPUs.
Source: GoogleNews
---

Title: Why context window matters in RAG
URL: https://example.com/context-window
Content: Understanding the trade-offs between large context windows and efficient retrieval augmented generation.
Source: Tavily
---`, nil
		},
	}
}

func main() {
	godotenv.Load(".env")

	dbURL := os.Getenv("DATABASE_URL")
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" || dbURL == "" {
		log.Fatal("DATABASE_URL and GROQ_API_KEY are required")
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/test_agent_mock.go <email_or_user_id>")
	}
	input := os.Args[1]

	// 1. Setup Store
	st, err := store.NewPostgresStore(context.Background(), dbURL)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Resolve User ID if email is provided
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

	// 2. Build Agent with Mocked Tool
	modelAdapter := groq.NewModel(groq.Config{APIKey: groqKey})

	getPrefsTool := tools.NewGetPreferencesTool(st)
	searchNewsMock := MockSearchNewsTool() // MOCKED
	storeArticlesTool := tools.NewStoreArticlesTool(st)

	myAgent, err := llmagent.New(llmagent.Config{
		Name:        "mock_daily_feed_agent",
		Model:       modelAdapter,
		Description: "A MOCKED agent for testing reasoning logic.",
		Instruction: prompts.AgentDailyFeed,
		Tools:       []tool.Tool{getPrefsTool, searchNewsMock, storeArticlesTool},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// 3. Setup Runner
	sessionSvc := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        "MockDailyFeed",
		Agent:          myAgent,
		SessionService: sessionSvc,
	})
	if err != nil {
		log.Fatalf("failed to create runner: %v", err)
	}

	// 4. Run
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	inputMsg := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(fmt.Sprintf("Generate daily feed for user_id: %s", userID)),
		},
	}

	log.Printf("Starting MOCKED Agent Run for User: %s", userID)

	next, stop := iter.Pull2(r.Run(ctx, userID, "mock-session", inputMsg, agent.RunConfig{}))
	defer stop()

	var finalSummary string
	for {
		event, err, ok := next()
		if !ok {
			break
		}
		if err != nil {
			log.Fatalf("Run error: %v", err)
		}

		if event.Content != nil {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					log.Printf("[Agent] %s", p.Text)
					finalSummary = p.Text
				}
			}
		}
	}

	log.Printf("\n--- MOCK TEST COMPLETE ---\nFINAL SUMMARY: %s", finalSummary)
}
