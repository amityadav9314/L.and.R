package settings

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

// Store defines the interface for settings persistence
type Store interface {
	GetSetting(ctx context.Context, key string) ([]byte, error)
	SetSetting(ctx context.Context, key string, value []byte, description string) error
}

// Service provides type-safe access to settings with in-memory caching
type Service struct {
	store         Store
	quotaLimits   QuotaLimits
	proAccessDays int
	mu            sync.RWMutex
}

// NewService creates a new settings service and loads settings from DB
func NewService(ctx context.Context, store Store) (*Service, error) {
	s := &Service{
		store:         store,
		quotaLimits:   DefaultQuotaLimits,
		proAccessDays: DefaultProAccessDays,
	}

	// Load from database
	if err := s.Refresh(ctx); err != nil {
		log.Printf("[Settings] Warning: Failed to load from DB, using defaults: %v", err)
	}

	return s, nil
}

// GetQuotaLimits returns the cached quota limits
func (s *Service) GetQuotaLimits() QuotaLimits {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.quotaLimits
}

// GetQuotaLimit returns the limit for a specific plan and resource
func (s *Service) GetQuotaLimit(plan, resource string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if plan == "pro" {
		return s.quotaLimits.Pro.GetLimit(resource)
	}
	return s.quotaLimits.Free.GetLimit(resource)
}

// GetProAccessDays returns the cached pro access days
func (s *Service) GetProAccessDays() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.proAccessDays
}

// Refresh reloads all settings from the database
func (s *Service) Refresh(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load quota limits
	data, err := s.store.GetSetting(ctx, string(KeyQuotaLimits))
	if err != nil {
		log.Printf("[Settings] Key '%s' not found in DB, using defaults", KeyQuotaLimits)
	} else {
		var limits QuotaLimits
		if err := json.Unmarshal(data, &limits); err != nil {
			log.Printf("[Settings] Failed to unmarshal '%s': %v", KeyQuotaLimits, err)
		} else {
			s.quotaLimits = limits
			log.Printf("[Settings] Loaded quota limits from DB: Free=%+v, Pro=%+v", limits.Free, limits.Pro)
		}
	}

	// Load pro access days
	data, err = s.store.GetSetting(ctx, string(KeyProAccessDays))
	if err != nil {
		log.Printf("[Settings] Key '%s' not found in DB, using default: %d", KeyProAccessDays, DefaultProAccessDays)
	} else {
		var days int
		if err := json.Unmarshal(data, &days); err != nil {
			log.Printf("[Settings] Failed to unmarshal '%s': %v", KeyProAccessDays, err)
		} else {
			s.proAccessDays = days
			log.Printf("[Settings] Loaded pro access days from DB: %d", days)
		}
	}

	return nil
}

// Seed inserts default values for any missing settings
func (s *Service) Seed(ctx context.Context) error {
	for _, key := range AllKeys() {
		_, err := s.store.GetSetting(ctx, string(key))
		if err == nil {
			continue
		}

		defaultVal := GetDefault(key)
		if defaultVal == nil {
			continue
		}

		data, err := json.Marshal(defaultVal)
		if err != nil {
			log.Printf("[Settings] Failed to marshal default for '%s': %v", key, err)
			continue
		}

		if err := s.store.SetSetting(ctx, string(key), data, KeyDescription(key)); err != nil {
			log.Printf("[Settings] Failed to seed '%s': %v", key, err)
			continue
		}

		log.Printf("[Settings] Seeded default for '%s'", key)
	}

	return s.Refresh(ctx)
}
