package feedagent

import (
	"context"
	"fmt"
	"iter"
	"log"
	"time"

	"github.com/amityadav/landr/internal/adk/tools"
	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/ai/models"
	"github.com/amityadav/landr/internal/search"
	"github.com/amityadav/landr/internal/store"
	adkmodel "github.com/amityadav/landr/pkg/adk/model"
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
	Store           *store.PostgresStore
	SearchProviders []search.SearchProvider
	AIProvider      ai.Provider
	GroqAPIKey      string
	CerebrasAPIKey  string
}

// NewFeedAgent creates a new Daily Feed Agent with V2 workflow
func NewFeedAgent(ctx context.Context, deps Dependencies) (agent.Agent, error) {
	// 1. Initialize fallback model (Groq → Cerebras on rate limit)
	modelName := models.TaskAgentDailyFeedModel
	log.Printf("[DailyFeedAgent] Initializing with model: %s (Groq primary, Cerebras fallback)", modelName)
	log.Printf("[DailyFeedAgent] Registered search providers: %d", len(deps.SearchProviders))

	modelAdapter, err := adkmodel.NewFallbackModel(deps.GroqAPIKey, deps.CerebrasAPIKey, modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback model: %w", err)
	}

	// 2. Define Tools using internal/adk/tools package
	allTools := getAllTools(deps)

	// 3. Create Agent with all V2 tools
	return llmagent.New(llmagent.Config{
		Name:        "daily_feed_agent_v2",
		Model:       modelAdapter,
		Description: "V2 Agent: Search → Batch Evaluate URLs → Store",
		Instruction: prompts.AgentDailyFeed,
		Tools:       allTools,
	})
}

func getAllTools(deps Dependencies) []tool.Tool {
	getPrefsTool := tools.NewGetPreferencesTool(deps.Store)
	searchNewsTool := tools.NewSearchNewsTool(deps.SearchProviders)
	evaluateURLsBatchTool := tools.NewEvaluateURLsBatchTool(deps.AIProvider)
	storeArticlesTool := tools.NewStoreArticlesTool(deps.Store)

	return []tool.Tool{
		getPrefsTool,
		searchNewsTool,
		evaluateURLsBatchTool,
		storeArticlesTool,
	}
}

// RunResult contains the outcome of an agent run
type RunResult struct {
	Summary string // The agent's final text response
	// TODO: Add StoredCount, SkippedCount once we implement shared state in tools
}

// Run executes the agent for a specific user and returns the result
func Run(ctx context.Context, deps Dependencies, userID string) (*RunResult, error) {
	myAgent, err := NewFeedAgent(ctx, deps)
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
	user, err := deps.Store.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("[DailyFeedAgent] Warning: could not fetch user email for logging: %v", err)
	}
	userEmail := "unknown"
	if user != nil {
		userEmail = user.Email
	}

	sessionID := fmt.Sprintf("%s-%s-%s", userID, userEmail, time.Now().Format("20060102-150405"))

	// Create session
	_, err = sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   "DailyFeed",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	inputMsg := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			genai.NewPartFromText(fmt.Sprintf("Generate daily feed for user_id: %s", userID)),
		},
	}

	// Execute Run
	log.Printf("[DailyFeedAgent] Starting V2 run for User: %s (%s) | Model: %s", userEmail, userID, models.TaskAgentDailyFeedModel)

	var finalResponse string

	// iter.Seq2 usage:
	next, stop := iter.Pull2(r.Run(ctx, userID, sessionID, inputMsg, agent.RunConfig{}))
	defer stop()

	finalResponse, err2 := processEvent(next, finalResponse)
	if err2 != nil {
		return nil, err2
	}

	log.Printf("[DailyFeedAgent] V2 run completed for user %s", userID)
	return &RunResult{Summary: finalResponse}, nil
}

func processEvent(next func() (*session.Event, error, bool), finalResponse string) (string, error) {
	for {
		event, err, ok := next()
		if !ok {
			break
		}
		if err != nil {
			log.Printf("[DailyFeedAgent] Error during run: %v", err)
			return "", err
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
	return finalResponse, nil
}
