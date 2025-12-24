package feedagent

import (
	"context"
	"fmt"
	"iter"
	"log"
	"time"

	"github.com/amityadav/landr/internal/adk/tools"
	"github.com/amityadav/landr/internal/serpapi"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/tavily"
	"github.com/amityadav/landr/pkg/adk/model/groq"
	"github.com/amityadav/landr/prompts"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// Dependencies holds the services needed by the agent
type Dependencies struct {
	Store      *store.PostgresStore
	Tavily     *tavily.Client
	SerpApi    *serpapi.Client
	GroqAPIKey string
}

// New creates a new Daily Feed Agent
func New(ctx context.Context, deps Dependencies) (agent.Agent, error) {
	// 1. Initialize custom Groq Model Adapter
	modelAdapter := groq.NewModel(groq.Config{
		APIKey: deps.GroqAPIKey,
		// Uses defaults: Groq endpoint and gpt-oss-120b
	})

	// 2. Define Tools using internal/adk/tools package
	getPrefsTool := tools.NewGetPreferencesTool(deps.Store)
	searchNewsTool := tools.NewSearchNewsTool(deps.Tavily, deps.SerpApi)
	storeArticlesTool := tools.NewStoreArticlesTool(deps.Store)

	// 3. Create Agent
	return llmagent.New(llmagent.Config{
		Name:        "daily_feed_agent",
		Model:       modelAdapter,
		Description: "An agent that curates daily AI/ML news.",
		Instruction: prompts.AgentDailyFeed,
		Tools:       []tool.Tool{getPrefsTool, searchNewsTool, storeArticlesTool},
	})
}

// RunResult contains the outcome of an agent run
type RunResult struct {
	Summary string // The agent's final text response
	// TODO: Add StoredCount, SkippedCount once we implement shared state in tools
}

// Run executes the agent for a specific user and returns the result
func Run(ctx context.Context, deps Dependencies, userID string) (*RunResult, error) {
	myAgent, err := New(ctx, deps)
	if err != nil {
		return nil, err
	}

	// Create InMemory Session Service
	sessionSvc := session.InMemoryService()

	// Create Runner
	r, err := runner.New(runner.Config{
		AppName:        "DailyFeed",
		Agent:          myAgent,
		SessionService: sessionSvc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	// Prepare Input
	sessionID := userID + "-" + time.Now().Format("20060102")
	inputMsg := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(fmt.Sprintf("Generate daily feed for user_id: %s", userID)),
		},
	}

	// Execute Run
	log.Printf("[DailyFeedAgent] Starting run for user %s", userID)

	var finalResponse string

	// iter.Seq2 usage:
	next, stop := iter.Pull2(r.Run(ctx, userID, sessionID, inputMsg, agent.RunConfig{}))
	defer stop()

	for {
		event, err, ok := next()
		if !ok {
			break
		}
		if err != nil {
			log.Printf("[DailyFeedAgent] Error during run: %v", err)
			return nil, err
		}

		// Log events and capture final response
		if event.Content != nil {
			for _, p := range event.Content.Parts {
				if p.Text != "" {
					log.Printf("[DailyFeedAgent] Event: %s", p.Text)
					// Capture the last text output as the response
					// In a multi-turn agent, we might want specifically the "model" final answer.
					// ADK events stream steps. The final one is usually the answer.
					finalResponse = p.Text
				}
			}
		}
	}

	log.Printf("[DailyFeedAgent] Run completed for user %s", userID)
	return &RunResult{Summary: finalResponse}, nil
}
