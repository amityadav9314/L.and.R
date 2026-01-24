package settings

// SettingKey represents a valid setting key
type SettingKey string

const (
	// KeyQuotaLimits stores daily quota limits per material type for Free and Pro plans
	KeyQuotaLimits SettingKey = "quota_limits"

	// KeyProAccessDays stores the default number of days for Pro subscription access
	KeyProAccessDays SettingKey = "pro_access_days"
)

// AllKeys returns all valid setting keys (for validation/seeding)
func AllKeys() []SettingKey {
	return []SettingKey{
		KeyQuotaLimits,
		KeyProAccessDays,
	}
}

// KeyDescription returns a human-readable description for a setting key
func KeyDescription(key SettingKey) string {
	switch key {
	case KeyQuotaLimits:
		return "Daily quota limits per material type for Free and Pro plans"
	case KeyProAccessDays:
		return "Default number of days for Pro subscription access"
	default:
		return ""
	}
}
