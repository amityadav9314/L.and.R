package quota

import "github.com/amityadav/landr/internal/store"

// Resource types
const (
	ResourceLinkImport = "link_import"
	ResourceTextImport = "text_import"
)

// Limits
const (
	FreeLinkLimit = 3
	FreeTextLimit = 10

	// Effectively unlimited for Pro
	ProLinkLimit = 50
	ProTextLimit = 100000
)

// GetLimit returns the limit for a resource based on the plan
func GetLimit(plan store.SubscriptionPlan, resource string) int {
	if plan == store.PlanPro {
		switch resource {
		case ResourceLinkImport:
			return ProLinkLimit
		case ResourceTextImport:
			return ProTextLimit
		}
	} else {
		// Default to Free
		switch resource {
		case ResourceLinkImport:
			return FreeLinkLimit
		case ResourceTextImport:
			return FreeTextLimit
		}
	}
	return 0
}
