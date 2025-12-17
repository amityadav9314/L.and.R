package core

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/amityadav/landr/internal/ai"
	"github.com/amityadav/landr/internal/scraper"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/youtube"
	"github.com/amityadav/landr/pkg/pb/learning"
)

type LearningCore struct {
	store   store.Store
	scraper *scraper.Scraper
	ai      ai.Provider
	youtube *youtube.TranscriptExtractor
}

func NewLearningCore(s store.Store, scraper *scraper.Scraper, aiProvider ai.Provider) *LearningCore {
	return &LearningCore{
		store:   s,
		scraper: scraper,
		ai:      aiProvider,
		youtube: youtube.NewTranscriptExtractor(),
	}
}

func (c *LearningCore) AddMaterial(ctx context.Context, userID, matType, content, imageData string, existingTags []string) (string, int32, string, []string, error) {
	log.Printf("[Core.AddMaterial] Starting - UserID: %s, Type: %s", userID, matType)

	// 1. Process Content based on type
	finalContent := content

	switch matType {
	case "LINK":
		log.Printf("[Core.AddMaterial] Scraping URL: %s", content)
		scraped, err := c.scraper.Scrape(content)
		if err != nil {
			log.Printf("[Core.AddMaterial] Scraping failed: %v", err)
			return "", 0, "", nil, fmt.Errorf("failed to scrape url: %w", err)
		}
		finalContent = scraped
		log.Printf("[Core.AddMaterial] Scraped content length: %d", len(finalContent))

	case "IMAGE":
		log.Printf("[Core.AddMaterial] Extracting text from image, base64 length: %d", len(imageData))
		if imageData == "" {
			return "", 0, "", nil, fmt.Errorf("image_data required for IMAGE type")
		}
		extractedText, err := c.ai.ExtractTextFromImage(imageData)
		if err != nil {
			log.Printf("[Core.AddMaterial] OCR extraction failed: %v", err)
			return "", 0, "", nil, fmt.Errorf("failed to extract text from image: %w", err)
		}
		finalContent = extractedText
		log.Printf("[Core.AddMaterial] OCR extracted text length: %d", len(finalContent))

	case "YOUTUBE":
		log.Printf("[Core.AddMaterial] Extracting YouTube transcript: %s", content)
		transcript, err := c.youtube.GetTranscript(ctx, content)
		if err != nil {
			log.Printf("[Core.AddMaterial] YouTube transcript failed: %v", err)
			return "", 0, "", nil, fmt.Errorf("failed to get youtube transcript: %w", err)
		}
		finalContent = transcript
		log.Printf("[Core.AddMaterial] YouTube transcript length: %d", len(finalContent))

	case "TEXT":
		log.Printf("[Core.AddMaterial] Using provided text content, length: %d", len(content))

	default:
		log.Printf("[Core.AddMaterial] Unknown type: %s, treating as TEXT", matType)
	}

	// 2. Fetch existing tags for AI context
	userTags, err := c.store.GetTags(ctx, userID)
	if err != nil {
		log.Printf("[Core.AddMaterial] Failed to fetch tags: %v", err)
	}

	// 3. Generate Flashcards + Summary in PARALLEL (MultiProvider races Groq vs Cerebras)
	log.Printf("[Core.AddMaterial] Starting AI generation with %s...", c.ai.Name())

	var title string
	var tags []string
	var cards []*learning.Flashcard
	var summary string
	var flashcardErr, summaryErr error

	// Run flashcards and summary in parallel - different providers won't conflict
	done := make(chan struct{}, 2)

	go func() {
		defer func() { done <- struct{}{} }()
		log.Printf("[Core.AddMaterial] Generating flashcards...")
		title, tags, cards, flashcardErr = c.ai.GenerateFlashcards(finalContent, userTags)
		if flashcardErr != nil {
			log.Printf("[Core.AddMaterial] Flashcard generation failed: %v", flashcardErr)
		} else {
			log.Printf("[Core.AddMaterial] Flashcards generated: %d cards", len(cards))
		}
	}()

	go func() {
		defer func() { done <- struct{}{} }()
		log.Printf("[Core.AddMaterial] Generating summary...")
		summary, summaryErr = c.ai.GenerateSummary(finalContent)
		if summaryErr != nil {
			log.Printf("[Core.AddMaterial] Summary generation failed: %v", summaryErr)
		} else {
			log.Printf("[Core.AddMaterial] Summary generated, length: %d", len(summary))
		}
	}()

	// Wait for both
	<-done
	<-done
	log.Printf("[Core.AddMaterial] AI generation complete")

	// Check for flashcard error (critical)
	if flashcardErr != nil {
		return "", 0, "", nil, fmt.Errorf("failed to generate flashcards: %w", flashcardErr)
	}
	// Summary error is non-critical - we can continue without it

	log.Printf("[Core.AddMaterial] AI generated Title: %s, Tags: %v, Cards: %d", title, tags, len(cards))

	// Use background context for DB operations - don't let client disconnect cancel saves
	// AI is already done, we MUST save the results even if client disconnects
	saveCtx := context.Background()

	// 4. Save Material with Title
	log.Printf("[Core.AddMaterial] Saving material to database...")
	materialID, err := c.store.CreateMaterial(saveCtx, userID, matType, finalContent, title)
	if err != nil {
		log.Printf("[Core.AddMaterial] Failed to save material: %v", err)
		return "", 0, "", nil, fmt.Errorf("failed to create material: %w", err)
	}
	log.Printf("[Core.AddMaterial] Material saved with ID: %s", materialID)

	// 5. Save Summary if generated
	if summary != "" && summaryErr == nil {
		if err := c.store.UpdateMaterialSummary(saveCtx, materialID, summary); err != nil {
			log.Printf("[Core.AddMaterial] Failed to save summary: %v", err)
			// Non-critical, continue
		} else {
			log.Printf("[Core.AddMaterial] Summary saved successfully")
		}
	}

	// 6. Save Tags and Link to Material
	var tagIDs []string
	for _, tagName := range tags {
		tagID, err := c.store.CreateTag(saveCtx, userID, tagName)
		if err != nil {
			log.Printf("[Core.AddMaterial] Failed to create tag %s: %v", tagName, err)
			continue
		}
		tagIDs = append(tagIDs, tagID)
	}

	if len(tagIDs) > 0 {
		if err := c.store.AddMaterialTags(saveCtx, materialID, tagIDs); err != nil {
			log.Printf("[Core.AddMaterial] Failed to link tags: %v", err)
		}
	}

	// 7. Save Flashcards
	if len(cards) > 0 {
		log.Printf("[Core.AddMaterial] Saving %d flashcards to database...", len(cards))
		if err := c.store.CreateFlashcards(saveCtx, materialID, cards); err != nil {
			log.Printf("[Core.AddMaterial] Failed to save flashcards: %v", err)
			return materialID, 0, title, tags, fmt.Errorf("failed to save flashcards: %w", err)
		}
		log.Printf("[Core.AddMaterial] Flashcards saved successfully")
	}

	log.Printf("[Core.AddMaterial] Complete - MaterialID: %s, Cards: %d", materialID, len(cards))
	return materialID, int32(len(cards)), title, tags, nil
}

func (c *LearningCore) DeleteMaterial(ctx context.Context, userID, materialID string) error {
	log.Printf("[Core.DeleteMaterial] Deleting material: %s for user: %s", materialID, userID)
	if err := c.store.SoftDeleteMaterial(ctx, userID, materialID); err != nil {
		log.Printf("[Core.DeleteMaterial] Failed: %v", err)
		return err
	}
	log.Printf("[Core.DeleteMaterial] Successfully deleted")
	return nil
}

func (c *LearningCore) GetDueFlashcards(ctx context.Context, userID, materialID string) ([]*learning.Flashcard, error) {
	log.Printf("[Core.GetDueFlashcards] Querying for userID: %s, materialID: %s", userID, materialID)
	cards, err := c.store.GetDueFlashcards(ctx, userID, materialID)
	if err != nil {
		log.Printf("[Core.GetDueFlashcards] Query failed: %v", err)
		return nil, err
	}
	log.Printf("[Core.GetDueFlashcards] Found %d cards", len(cards))
	return cards, nil
}

func (c *LearningCore) GetDueMaterials(ctx context.Context, userID string, page, pageSize int32) ([]*learning.MaterialSummary, int32, error) {
	log.Printf("[Core.GetDueMaterials] Querying for userID: %s, page: %d, pageSize: %d", userID, page, pageSize)
	materials, totalCount, err := c.store.GetDueMaterials(ctx, userID, page, pageSize)
	if err != nil {
		log.Printf("[Core.GetDueMaterials] Query failed: %v", err)
		return nil, 0, err
	}
	log.Printf("[Core.GetDueMaterials] Found %d materials (total: %d)", len(materials), totalCount)
	return materials, totalCount, nil
}

func (c *LearningCore) CompleteReview(ctx context.Context, flashcardID string) error {
	log.Printf("[Core.CompleteReview] Updating flashcard: %s", flashcardID)

	// Fetch the current flashcard to get its stage
	card, err := c.store.GetFlashcard(ctx, flashcardID)
	if err != nil {
		log.Printf("[Core.CompleteReview] Failed to get flashcard: %v", err)
		return fmt.Errorf("failed to get flashcard: %w", err)
	}

	// Implement SRS logic: increment stage and calculate next review time
	// Stage 0: New card -> 1 day
	// Stage 1: 1 day -> 3 days
	// Stage 2: 3 days -> 7 days
	// Stage 3: 7 days -> 15 days
	// Stage 4: 15 days -> 30 days
	// Stage 5+: 30 days (max)

	currentStage := card.Stage
	nextStage := currentStage + 1

	// Calculate next review interval based on new stage
	var intervalDays int
	switch nextStage {
	case 1:
		intervalDays = 1
	case 2:
		intervalDays = 3
	case 3:
		intervalDays = 7
	case 4:
		intervalDays = 15
	default:
		// Stage 5 and above: 30 days
		intervalDays = 30
		if nextStage > 5 {
			nextStage = 5 // Cap at stage 5
		}
	}

	nextReviewAt := time.Now().Add(time.Duration(intervalDays) * 24 * time.Hour)

	log.Printf("[Core.CompleteReview] Advancing from stage %d to %d (next review in %d days)",
		currentStage, nextStage, intervalDays)

	err = c.store.UpdateFlashcard(ctx, flashcardID, nextStage, nextReviewAt)
	if err != nil {
		log.Printf("[Core.CompleteReview] Update failed: %v", err)
		return err
	}

	log.Printf("[Core.CompleteReview] Updated successfully to stage %d", nextStage)
	return nil
}

func (c *LearningCore) FailReview(ctx context.Context, flashcardID string) error {
	log.Printf("[Core.FailReview] Failing flashcard: %s", flashcardID)

	// Fetch the current flashcard to get its stage
	card, err := c.store.GetFlashcard(ctx, flashcardID)
	if err != nil {
		log.Printf("[Core.FailReview] Failed to get flashcard: %v", err)
		return fmt.Errorf("failed to get flashcard: %w", err)
	}

	// Decrease stage by 1, minimum 0
	currentStage := card.Stage
	nextStage := currentStage - 1
	if nextStage < 0 {
		nextStage = 0
	}

	// Reset to review in 1 day (back to basics)
	nextReviewAt := time.Now().Add(24 * time.Hour)

	log.Printf("[Core.FailReview] Decreasing from stage %d to %d (next review in 1 day)",
		currentStage, nextStage)

	err = c.store.UpdateFlashcard(ctx, flashcardID, nextStage, nextReviewAt)
	if err != nil {
		log.Printf("[Core.FailReview] Update failed: %v", err)
		return err
	}

	log.Printf("[Core.FailReview] Updated successfully to stage %d", nextStage)
	return nil
}

func (c *LearningCore) UpdateFlashcard(ctx context.Context, flashcardID, question, answer string) error {
	log.Printf("[Core.UpdateFlashcard] Updating flashcard: %s", flashcardID)
	if err := c.store.UpdateFlashcardContent(ctx, flashcardID, question, answer); err != nil {
		log.Printf("[Core.UpdateFlashcard] Failed: %v", err)
		return err
	}
	log.Printf("[Core.UpdateFlashcard] Successfully updated")
	return nil
}

func (c *LearningCore) GetAllTags(ctx context.Context, userID string) ([]string, error) {
	return c.store.GetTags(ctx, userID)
}

func (c *LearningCore) GetNotificationStatus(ctx context.Context, userID string) (int32, bool, error) {
	log.Printf("[Core.GetNotificationStatus] Getting notification status for userID: %s", userID)

	count, err := c.store.GetDueFlashcardsCount(ctx, userID)
	if err != nil {
		log.Printf("[Core.GetNotificationStatus] Failed to get count: %v", err)
		return 0, false, err
	}

	hasDue := count > 0
	log.Printf("[Core.GetNotificationStatus] User has %d due flashcards", count)
	return count, hasDue, nil
}

func (c *LearningCore) GetMaterialSummary(ctx context.Context, userID, materialID string) (string, string, error) {
	log.Printf("[Core.GetMaterialSummary] Getting summary for materialID: %s, userID: %s", materialID, userID)

	// 1. Fetch material content and existing summary
	content, summary, title, err := c.store.GetMaterialContent(ctx, userID, materialID)
	if err != nil {
		log.Printf("[Core.GetMaterialSummary] Failed to get material: %v", err)
		return "", "", fmt.Errorf("failed to get material: %w", err)
	}

	// 2. If summary exists, return it
	if summary != "" {
		log.Printf("[Core.GetMaterialSummary] Returning existing summary, length: %d", len(summary))
		return summary, title, nil
	}

	// 3. Generate summary via AI
	log.Printf("[Core.GetMaterialSummary] No summary found, generating via AI...")
	summary, err = c.ai.GenerateSummary(content)
	if err != nil {
		log.Printf("[Core.GetMaterialSummary] AI generation failed: %v", err)
		return "", title, fmt.Errorf("failed to generate summary: %w", err)
	}

	// 4. Save summary to database
	if err := c.store.UpdateMaterialSummary(ctx, materialID, summary); err != nil {
		log.Printf("[Core.GetMaterialSummary] Failed to save summary: %v", err)
		// Continue - we can still return the generated summary
	}

	log.Printf("[Core.GetMaterialSummary] Summary generated and saved, length: %d", len(summary))
	return summary, title, nil
}
