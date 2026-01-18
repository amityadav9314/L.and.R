package core

import (
	"context"
	"fmt"
	"time"

	"github.com/amityadav/landr/internal/store"
	"github.com/amityadav/landr/internal/token"
	"github.com/amityadav/landr/pkg/pb/auth"
	"google.golang.org/api/idtoken"
)

type AuthCore struct {
	store        store.Store
	tokenManager *token.Manager
	clientID     string
}

func NewAuthCore(s store.Store, tm *token.Manager, clientID string) *AuthCore {
	return &AuthCore{
		store:        s,
		tokenManager: tm,
		clientID:     clientID,
	}
}

// LoginUser handles the business logic for user login.
// It verifies the token, syncs the user to DB, and issues a JWT.
func (c *AuthCore) LoginUser(ctx context.Context, googleToken string) (*auth.UserProfile, string, error) {
	// 1. Verify Google Token
	payload, err := c.verifyGoogleToken(ctx, googleToken)
	if err != nil {
		return nil, "", fmt.Errorf("verify google token: %w", err)
	}

	// 2. Create or Update User
	user, err := c.store.CreateUser(ctx,
		payload.Claims["email"].(string),
		payload.Claims["name"].(string),
		payload.Subject,
		payload.Claims["picture"].(string),
	)
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	if user.IsBlocked {
		return nil, "", fmt.Errorf("account blocked")
	}

	// 3. Generate Session Token
	sessionToken, err := c.tokenManager.NewJWT(user.Id, 24*time.Hour*30)
	if err != nil {
		return nil, "", fmt.Errorf("generate session token: %w", err)
	}

	return user, sessionToken, nil
}

func (c *AuthCore) verifyGoogleToken(ctx context.Context, token string) (*idtoken.Payload, error) {
	payload, err := idtoken.Validate(ctx, token, c.clientID)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
