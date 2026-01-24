package settings

// DefaultQuotaLimits provides the default quota limits
// Used for seeding and fallback when DB is unavailable
var DefaultQuotaLimits = QuotaLimits{
	Free: TypeLimits{
		Link:    3,
		Text:    10,
		Image:   5,
		YouTube: 3,
	},
	Pro: TypeLimits{
		Link:    50,
		Text:    100000,
		Image:   100,
		YouTube: 50,
	},
}

// DefaultProAccessDays is the default number of days for Pro subscription
const DefaultProAccessDays = 30

// GetDefault returns the default value for a setting key
func GetDefault(key SettingKey) interface{} {
	switch key {
	case KeyQuotaLimits:
		return DefaultQuotaLimits
	case KeyProAccessDays:
		return DefaultProAccessDays
	default:
		return nil
	}
}
