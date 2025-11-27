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

	materialID, count, title, tags, err := s.core.AddMaterial(ctx, userID, req.Type, req.Content, req.ExistingTags)
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

func (s *LearningService) GetDueMaterials(ctx context.Context, _ *emptypb.Empty) (*learning.GetDueMaterialsResponse, error) {
	// Extract user ID from context (set by auth interceptor)
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		log.Printf("[GetDueMaterials] ERROR: Failed to get user ID: %v", err)
		return nil, err
	}
	log.Printf("[GetDueMaterials] Fetching materials for userID: %s", userID)

	materials, err := s.core.GetDueMaterials(ctx, userID)
	if err != nil {
		log.Printf("[GetDueMaterials] ERROR: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get due materials: %v", err)
	}

	log.Printf("[GetDueMaterials] SUCCESS - Found %d materials", len(materials))
	return &learning.GetDueMaterialsResponse{
		Materials: materials,
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
