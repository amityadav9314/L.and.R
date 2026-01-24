package quota

import (
	"context"
	"log"

	"github.com/amityadav/landr/internal/middleware"
	"github.com/amityadav/landr/internal/settings"
	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/pkg/pb/learning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Interceptor struct {
	store    store.Store
	settings *settings.Service
}

func NewInterceptor(s store.Store, settingsSvc *settings.Service) *Interceptor {
	return &Interceptor{
		store:    s,
		settings: settingsSvc,
	}
}

func (i *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 1. Identify if this method needs quota check
		resource := i.getResourceForRequest(info.FullMethod, req)
		if resource == "" {
			// No quota check needed
			return handler(ctx, req)
		}

		// 2. Get User ID
		userID, err := middleware.GetUserID(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		// 3. Get User's Subscription
		sub, err := i.store.GetSubscription(ctx, userID)
		if err != nil {
			log.Printf("Failed to get subscription for user %s: %v", userID, err)
			return nil, status.Error(codes.Internal, "failed to check subscription status")
		}

		// 4. Check Quota
		limit := i.getLimit(sub.Plan, resource)
		allowed, err := i.store.CheckQuota(ctx, userID, resource, limit)
		if err != nil {
			log.Printf("Failed to check quota for user %s: %v", userID, err)
			return nil, status.Error(codes.Internal, "failed to check quota")
		}

		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted,
				"You've reached your daily limit of %d %s. Upgrade to Pro for more!",
				limit, ResourceDisplayName(resource))
		}

		// 5. Execute Handler
		resp, err := handler(ctx, req)

		// 6. If successful, increment quota
		if err == nil {
			if incErr := i.store.IncrementQuota(ctx, userID, resource); incErr != nil {
				log.Printf("Failed to increment quota for user %s: %v", userID, incErr)
			}
		}

		return resp, err
	}
}

func (i *Interceptor) getResourceForRequest(method string, req interface{}) string {
	// Only checking AddMaterial for now
	if method == "/learning.LearningService/AddMaterial" {
		if r, ok := req.(*learning.AddMaterialRequest); ok {
			switch r.Type {
			case "LINK":
				return ResourceLinkImport
			case "IMAGE":
				return ResourceImageImport
			case "YOUTUBE":
				return ResourceYoutubeImport
			default:
				return ResourceTextImport
			}
		}
	}
	return ""
}

func (i *Interceptor) getLimit(plan store.SubscriptionPlan, resource string) int {
	planStr := "free"
	if plan == store.PlanPro {
		planStr = "pro"
	}
	return i.settings.GetQuotaLimit(planStr, resource)
}
