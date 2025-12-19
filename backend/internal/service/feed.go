package service

import (
	"context"

	"github.com/amityadav/landr/internal/core"
	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/pkg/pb/feed"
	"google.golang.org/protobuf/types/known/emptypb"
)

// FeedService implements the FeedServiceServer gRPC interface
type FeedService struct {
	feed.UnimplementedFeedServiceServer
	core *core.FeedCore
}

// NewFeedService creates a new FeedService
func NewFeedService(c *core.FeedCore) *FeedService {
	return &FeedService{core: c}
}

// UpdateFeedPreferences implements FeedServiceServer.UpdateFeedPreferences
func (s *FeedService) UpdateFeedPreferences(ctx context.Context, req *feed.UpdateFeedPreferencesRequest) (*emptypb.Empty, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	err = s.core.UpdateFeedPreferences(ctx, userID, req.InterestPrompt, req.FeedEnabled)
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
