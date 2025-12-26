package feedagent_test

import (
	"context"
	"fmt"
	"iter"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/amityadav/landr/internal/adk/tools"
	"github.com/amityadav/landr/internal/ai/models"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/adk/model/groq"
	"github.com/amityadav/landr/prompts"
	"github.com/joho/godotenv"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// MockSearchNewsArgs matches the real SearchNewsArgs
type MockSearchNewsArgs struct {
	Queries []string `json:"queries"`
}

// MockSearchNewsResult matches the real SearchNewsResult
type MockSearchNewsResult struct {
	Articles string `json:"articles"`
}

// MockSearchNewsTool returns dummy articles without calling real APIs
func MockSearchNewsTool() tool.Tool {
	handler := func(ctx tool.Context, args MockSearchNewsArgs) (MockSearchNewsResult, error) {
		log.Printf("[MOCK] search_news called with %d queries: %v", len(args.Queries), args.Queries)

		return MockSearchNewsResult{Articles: `Found 5 articles:

Title: Mock Article - AI Agents are the future of software
URL: https://example.com/ai-agents
Content: A deep dive into how autonomous agents like AutoGPT and specialized ADK agents are changing the landscape of software development.
Source: Tavily
---

Title: Mock Article - New Groq LPU benchmarks released
URL: https://example.com/groq-benchmarks
Snippet: Groq's latest LPU shows 10x performance increase in token generation for Llama 3 models compared to traditional GPUs.
Source: GoogleNews
---

Title: Mock Article - Why context window matters in RAG
URL: https://example.com/context-window
Content: Understanding the trade-offs between large context windows and efficient retrieval augmented generation.
Source: Tavily
---

Title: Mock Article - Open Source LLMs gaining traction
URL: https://example.com/open-llms
Snippet: New open weights models from Meta and Mistral are challenging proprietary solutions.
Source: GoogleNews
---

Title: Mock Article - Vector databases comparison 2024
URL: https://example.com/vector-db
Content: Comparing Pinecone, Weaviate, and ChromaDB for production use cases.
Source: Tavily
---`}, nil
	}

	t, err := functiontool.New(functiontool.Config{
		Name:        "search_news",
		Description: prompts.ToolSearchNewsDesc,
	}, handler)
	if err != nil {
		log.Fatalf("Failed to create mock search_news tool: %v", err)
	}
	return t
}

func TestAgentWithMockedSearch(t *testing.T) {
	// Load env from backend/.env (relative to internal/adk/feedagent/)
	godotenv.Load("../../../.env")

	dbURL := os.Getenv("DATABASE_URL")
	groqKey := os.Getenv("GROQ_API_KEY")
	testEmail := os.Getenv("TEST_USER_EMAIL") // Set this in .env for testing

	if groqKey == "" || dbURL == "" {
		t.Skip("DATABASE_URL and GROQ_API_KEY required for this test")
	}
	if testEmail == "" {
		t.Skip("TEST_USER_EMAIL required for this test")
	}

	// 1. Setup Real Store
	st, err := store.NewPostgresStore(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	// 2. Resolve User ID from email
	userID, err := st.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to find user with email %s: %v", testEmail, err)
	}
	t.Logf("Testing with UserID: %s", userID)

	// 3. Build Agent with REAL Groq, REAL DB	// Create Groq Model Adapter (Real LLM, mocked tools)
	groqModel := groq.NewModel(groq.Config{
		APIKey:    os.Getenv("GROQ_API_KEY"),
		ModelName: models.TaskAgentDailyFeedModel, // Use robust model for test
	})
	getPrefsTool := tools.NewGetPreferencesTool(st)
	searchNewsMock := MockSearchNewsTool() // <-- MOCKED
	storeArticlesTool := tools.NewStoreArticlesTool(st)

	myAgent, err := llmagent.New(llmagent.Config{
		Name:        "daily_feed_agent",
		Model:       groqModel,
		Description: "Daily Feed Agent (with mocked search for testing)",
		Instruction: prompts.AgentDailyFeed,
		Tools:       []tool.Tool{getPrefsTool, searchNewsMock, storeArticlesTool},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// 4. Setup Runner
	sessionSvc := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        "DailyFeed",
		Agent:          myAgent,
		SessionService: sessionSvc,
	})
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	// 5. Create Session (required by ADK before running)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	sessionID := userID + "-test-" + time.Now().Format("20060102150405")

	_, err = sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   "DailyFeed",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 6. Run Agent
	inputMsg := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(fmt.Sprintf("Generate daily feed for user_id: %s", userID)),
		},
	}

	t.Logf("Starting agent run (Session: %s)...", sessionID)

	next, stop := iter.Pull2(r.Run(ctx, userID, sessionID, inputMsg, agent.RunConfig{}))
	defer stop()

	var finalSummary string
	for {
		event, err, ok := next()
		if !ok {
			break
		}
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}

		if event.Content != nil {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					t.Logf("[Agent] %s", p.Text)
					finalSummary = p.Text
				}
			}
		}
	}

	if finalSummary == "" {
		t.Error("Agent returned empty summary")
	} else {
		t.Logf("SUCCESS! Final Summary: %s", finalSummary)
	}

	// Verify articles were stored
	if strings.Contains(finalSummary, "stored") || strings.Contains(finalSummary, "article") {
		t.Log("Agent appears to have stored articles successfully")
	}
}
