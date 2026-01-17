package service

import (
	"context"

	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/pb/feed"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// FeedService implements the FeedServiceServer gRPC interface
type FeedService struct {
	feed.UnimplementedFeedServiceServer
	core  *core.FeedCore
	store store.Store
}

// NewFeedService creates a new FeedService
func NewFeedService(c *core.FeedCore, s store.Store) *FeedService {
	return &FeedService{core: c, store: s}
}

// UpdateFeedPreferences implements FeedServiceServer.UpdateFeedPreferences
func (s *FeedService) UpdateFeedPreferences(ctx context.Context, req *feed.UpdateFeedPreferencesRequest) (*emptypb.Empty, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 1. Check if user is trying to set custom prompts
	hasCustomPrompts := req.InterestPrompt != "" || req.FeedEvalPrompt != ""

	if hasCustomPrompts {
		// 2. Check Subscription
		sub, err := s.store.GetSubscription(ctx, userID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check subscription")
		}

		if sub.Plan != store.PlanPro {
			return nil, status.Error(codes.PermissionDenied, "Customizing feed prompts is a Pro feature. Please upgrade.")
		}
	}

	err = s.core.UpdateFeedPreferences(ctx, userID, req.InterestPrompt, req.FeedEvalPrompt, req.FeedEnabled)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetFeedPreferences implements FeedServiceServer.GetFeedPreferences
func (s *FeedService) GetFeedPreferences(ctx context.Context, _ *emptypb.Empty) (*feed.FeedPreferencesResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	return s.core.GetFeedPreferences(ctx, userID)
}

// GetDailyFeed implements FeedServiceServer.GetDailyFeed
func (s *FeedService) GetDailyFeed(ctx context.Context, req *feed.GetDailyFeedRequest) (*feed.GetDailyFeedResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	return s.core.GetDailyFeed(ctx, userID, req.Date)
}

// GetFeedCalendarStatus implements FeedServiceServer.GetFeedCalendarStatus
func (s *FeedService) GetFeedCalendarStatus(ctx context.Context, req *feed.GetFeedCalendarStatusRequest) (*feed.GetFeedCalendarStatusResponse, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	return s.core.GetFeedCalendarStatus(ctx, userID, req.Month)
}
