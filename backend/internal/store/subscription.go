package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type SubscriptionPlan string
type SubscriptionStatus string

const (
	PlanFree SubscriptionPlan = "FREE"
	PlanPro  SubscriptionPlan = "PRO"

	StatusActive    SubscriptionStatus = "ACTIVE"
	StatusPastDue   SubscriptionStatus = "PAST_DUE"
	StatusCancelled SubscriptionStatus = "CANCELLED"
	StatusTrialing  SubscriptionStatus = "TRIALING"
)

type Subscription struct {
	UserID                 string
	Plan                   SubscriptionPlan
	Status                 SubscriptionStatus
	CurrentPeriodEnd       *time.Time
	RazorpaySubscriptionID string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// GetSubscription retrieves a user's subscription
func (s *PostgresStore) GetSubscription(ctx context.Context, userID string) (*Subscription, error) {
	query := `
		SELECT plan, status, current_period_end, razorpay_subscription_id, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1
	`
	var sub Subscription
	sub.UserID = userID
	var plan, status string
	var rzpID *string // Use pointer for NULL handling in pgx scan if flexible, or *string

	// Pgx scan handles nil for *time.Time and *string
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&plan, &status, &sub.CurrentPeriodEnd, &rzpID, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		// Return default FREE subscription if none exists
		return &Subscription{
			UserID: userID,
			Plan:   PlanFree,
			Status: StatusActive,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	sub.Plan = SubscriptionPlan(plan)
	sub.Status = SubscriptionStatus(status)
	if rzpID != nil {
		sub.RazorpaySubscriptionID = *rzpID
	}
	return &sub, nil
}

// UpsertSubscription creates or updates a subscription
func (s *PostgresStore) UpsertSubscription(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (user_id, plan, status, current_period_end, razorpay_subscription_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			plan = EXCLUDED.plan,
			status = EXCLUDED.status,
			current_period_end = EXCLUDED.current_period_end,
			razorpay_subscription_id = EXCLUDED.razorpay_subscription_id,
			updated_at = NOW()
	`
	_, err := s.db.Exec(ctx, query,
		sub.UserID, sub.Plan, sub.Status, sub.CurrentPeriodEnd, sub.RazorpaySubscriptionID,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}
	return nil
}

// CheckQuota checks if a user has exceeded their daily limit for a resource
func (s *PostgresStore) CheckQuota(ctx context.Context, userID, resource string, limit int) (bool, error) {
	// Transaction to ensure atomicity
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// distinct logic for reset:
	// If last_reset_at < CURRENT_DATE, count = 0, last_reset_at = CURRENT_DATE
	resetQuery := `
		INSERT INTO usage_quotas (user_id, resource, count, last_reset_at)
		VALUES ($1, $2, 0, CURRENT_DATE)
		ON CONFLICT (user_id, resource) DO UPDATE SET
			count = CASE WHEN usage_quotas.last_reset_at < CURRENT_DATE THEN 0 ELSE usage_quotas.count END,
			last_reset_at = CURRENT_DATE
		RETURNING count
	`
	var currentCount int
	err = tx.QueryRow(ctx, resetQuery, userID, resource).Scan(&currentCount)
	if err != nil {
		return false, fmt.Errorf("failed to check/reset quota: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return currentCount < limit, nil
}

// IncrementQuota increments the usage count
func (s *PostgresStore) IncrementQuota(ctx context.Context, userID, resource string) error {
	query := `
		INSERT INTO usage_quotas (user_id, resource, count, last_reset_at)
		VALUES ($1, $2, 1, CURRENT_DATE)
		ON CONFLICT (user_id, resource) DO UPDATE SET
			count = usage_quotas.count + 1,
			last_reset_at = CURRENT_DATE
	`
	_, err := s.db.Exec(ctx, query, userID, resource)
	if err != nil {
		return fmt.Errorf("failed to increment quota: %w", err)
	}
	return nil
}

// GetUsage returns current usage
func (s *PostgresStore) GetUsage(ctx context.Context, userID, resource string) (int, error) {
	query := `
		SELECT count, last_reset_at FROM usage_quotas WHERE user_id = $1 AND resource = $2
	`
	var count int
	var lastReset time.Time
	err := s.db.QueryRow(ctx, query, userID, resource).Scan(&count, &lastReset)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	if lastReset.Before(today) {
		return 0, nil
	}

	return count, nil
}
