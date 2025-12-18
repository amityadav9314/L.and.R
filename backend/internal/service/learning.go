package service

import (
	"context"
	"log"

	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/pkg/pb/learning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type LearningService struct {
	learning.UnimplementedLearningServiceServer
	core *core.LearningCore
}

func NewLearningService(c *core.LearningCore) *LearningService {
	return &LearningService{
		core: c,
	}
}

func (s *LearningService) AddMaterial(ctx context.Context, req *learning.AddMaterialRequest) (*learning.AddMaterialResponse, error) {
	log.Printf("[AddMaterial] Received request - Type: %s, Content length: %d", req.Type, len(req.Content))

	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[AddMaterial] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}
	log.Printf("[AddMaterial] Using userID: %s", userID)

	materialID, count, title, tags, err := s.core.AddMaterial(ctx, userID, req.Type, req.Content, req.ImageData, req.ExistingTags)
	if err != nil {
		log.Printf("[AddMaterial] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to add material: %v", err)
	}

	log.Printf("[AddMaterial] SUCCESS - MaterialID: %s, Flashcards created: %d", materialID, count)
	return &learning.AddMaterialResponse{
		MaterialId:        materialID,
		FlashcardsCreated: count,
		Title:             title,
		Tags:              tags,
	}, nil
}

func (s *LearningService) DeleteMaterial(ctx context.Context, req *learning.DeleteMaterialRequest) (*emptypb.Empty, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[DeleteMaterial] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}
	log.Printf("[DeleteMaterial] Deleting material: %s for user: %s", req.MaterialId, userID)

	if err := s.core.DeleteMaterial(ctx, userID, req.MaterialId); err != nil {
		log.Printf("[DeleteMaterial] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to delete material: %v", err)
	}

	log.Printf("[DeleteMaterial] SUCCESS")
	return &emptypb.Empty{}, nil
}

func (s *LearningService) GetDueFlashcards(ctx context.Context, req *learning.GetDueFlashcardsRequest) (*learning.FlashcardList, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetDueFlashcards] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}
	log.Printf("[GetDueFlashcards] Fetching flashcards for userID: %s, materialID: %s", userID, req.MaterialId)

	cards, err := s.core.GetDueFlashcards(ctx, userID, req.MaterialId)
	if err != nil {
		log.Printf("[GetDueFlashcards] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get due flashcards: %v", err)
	}

	log.Printf("[GetDueFlashcards] SUCCESS - Found %d flashcards", len(cards))
	return &learning.FlashcardList{
		Flashcards: cards,
	}, nil
}

func (s *LearningService) GetDueMaterials(ctx context.Context, req *learning.GetDueMaterialsRequest) (*learning.GetDueMaterialsResponse, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetDueMaterials] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}

	// Set default values if not provided
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10 // Default page size
	}

	log.Printf("[GetDueMaterials] Fetching materials for userID: %s, page: %d, pageSize: %d, search: %s, tags: %v, onlyDue: %v", userID, page, pageSize, req.SearchQuery, req.Tags, req.OnlyDue)

	materials, totalCount, err := s.core.GetDueMaterials(ctx, userID, page, pageSize, req.SearchQuery, req.Tags, req.OnlyDue)
	if err != nil {
		log.Printf("[GetDueMaterials] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get due materials: %v", err)
	}

	// Calculate total pages
	totalPages := (totalCount + pageSize - 1) / pageSize

	log.Printf("[GetDueMaterials] SUCCESS - Found %d materials (page %d/%d, total: %d)", len(materials), page, totalPages, totalCount)
	return &learning.GetDueMaterialsResponse{
		Materials:  materials,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *LearningService) CompleteReview(ctx context.Context, req *learning.CompleteReviewRequest) (*emptypb.Empty, error) {
	log.Printf("[CompleteReview] Completing review for flashcardID: %s", req.FlashcardId)

	if err := s.core.CompleteReview(ctx, req.FlashcardId); err != nil {
		log.Printf("[CompleteReview] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to complete review: %v", err)
	}

	log.Printf("[CompleteReview] SUCCESS")
	return &emptypb.Empty{}, nil
}

func (s *LearningService) FailReview(ctx context.Context, req *learning.FailReviewRequest) (*emptypb.Empty, error) {
	log.Printf("[FailReview] Failing review for flashcardID: %s", req.FlashcardId)

	if err := s.core.FailReview(ctx, req.FlashcardId); err != nil {
		log.Printf("[FailReview] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to fail review: %v", err)
	}

	log.Printf("[FailReview] SUCCESS")
	return &emptypb.Empty{}, nil
}

func (s *LearningService) UpdateFlashcard(ctx context.Context, req *learning.UpdateFlashcardRequest) (*emptypb.Empty, error) {
	log.Printf("[UpdateFlashcard] Updating flashcardID: %s", req.FlashcardId)

	if err := s.core.UpdateFlashcard(ctx, req.FlashcardId, req.Question, req.Answer); err != nil {
		log.Printf("[UpdateFlashcard] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to update flashcard: %v", err)
	}

	log.Printf("[UpdateFlashcard] SUCCESS")
	return &emptypb.Empty{}, nil
}

func (s *LearningService) GetAllTags(ctx context.Context, _ *emptypb.Empty) (*learning.GetAllTagsResponse, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetAllTags] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}

	log.Printf("[GetAllTags] Fetching all tags for userID: %s", userID)

	tags, err := s.core.GetAllTags(ctx, userID)
	if err != nil {
		log.Printf("[GetAllTags] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get tags: %v", err)
	}

	log.Printf("[GetAllTags] SUCCESS - Found %d unique tags", len(tags))
	return &learning.GetAllTagsResponse{
		Tags: tags,
	}, nil
}

func (s *LearningService) GetNotificationStatus(ctx context.Context, _ *emptypb.Empty) (*learning.NotificationStatusResponse, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetNotificationStatus] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}

	log.Printf("[GetNotificationStatus] Fetching notification status for userID: %s", userID)

	count, hasDue, err := s.core.GetNotificationStatus(ctx, userID)
	if err != nil {
		log.Printf("[GetNotificationStatus] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get notification status: %v", err)
	}

	log.Printf("[GetNotificationStatus] SUCCESS - %d due flashcards", count)
	return &learning.NotificationStatusResponse{
		DueFlashcardsCount: count,
		HasDueMaterials:    hasDue,
	}, nil
}

func (s *LearningService) GetMaterialSummary(ctx context.Context, req *learning.GetMaterialSummaryRequest) (*learning.GetMaterialSummaryResponse, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetMaterialSummary] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}

	log.Printf("[GetMaterialSummary] Fetching summary for materialID: %s, userID: %s", req.MaterialId, userID)

	summary, title, err := s.core.GetMaterialSummary(ctx, userID, req.MaterialId)
	if err != nil {
		log.Printf("[GetMaterialSummary] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get material summary: %v", err)
	}

	log.Printf("[GetMaterialSummary] SUCCESS - Summary length: %d", len(summary))
	return &learning.GetMaterialSummaryResponse{
		Summary: summary,
		Title:   title,
	}, nil
}
