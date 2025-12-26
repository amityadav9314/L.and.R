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
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/search"
	"github.com/amityadav/landr/internal/store"
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
	Store           *store.PostgresStore
	SearchProviders []search.SearchProvider // Changed: Use interface slice
	Scraper         *scraper.Scraper
	AIProvider      ai.Provider
	GroqAPIKey      string
}

// New creates a new Daily Feed Agent with V2 workflow
func New(ctx context.Context, deps Dependencies) (agent.Agent, error) {
	// 1. Initialize custom Groq Model Adapter
	modelName := models.TaskAgentDailyFeedModel
	log.Printf("[DailyFeedAgent] Initializing with model: %s", modelName)
	log.Printf("[DailyFeedAgent] Registered search providers: %d", len(deps.SearchProviders))

	modelAdapter := groq.NewModel(groq.Config{
		APIKey:    deps.GroqAPIKey,
		ModelName: modelName,
	})

	// 2. Define Tools using internal/adk/tools package
	getPrefsTool := tools.NewGetPreferencesTool(deps.Store)
	searchNewsTool := tools.NewSearchNewsTool(deps.SearchProviders) // Pass provider slice
	scrapeContentTool := tools.NewScrapeContentTool(deps.Scraper)
	summarizeContentTool := tools.NewSummarizeContentTool(deps.AIProvider)
	evaluateArticleTool := tools.NewEvaluateArticleTool(deps.AIProvider)
	storeArticlesTool := tools.NewStoreArticlesTool(deps.Store)

	// 3. Create Agent with all V2 tools
	return llmagent.New(llmagent.Config{
		Name:        "daily_feed_agent_v2",
		Model:       modelAdapter,
		Description: "V2 Agent: Scrape → Summarize → Evaluate → Store",
		Instruction: prompts.AgentDailyFeed,
		Tools: []tool.Tool{
			getPrefsTool,
			searchNewsTool,
			scrapeContentTool,
			summarizeContentTool,
			evaluateArticleTool,
			storeArticlesTool,
		},
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

	log.Printf("[DailyFeedAgent] V2 run completed for user %s", userID)
	return &RunResult{Summary: finalResponse}, nil
}
