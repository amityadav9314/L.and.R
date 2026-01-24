package settings

// TypeLimits defines daily limits for each material type
type TypeLimits struct {
	Link    int `json:"link"`
	Text    int `json:"text"`
	Image   int `json:"image"`
	YouTube int `json:"youtube"`
}

// QuotaLimits defines daily quota limits per subscription plan
type QuotaLimits struct {
	Free TypeLimits `json:"free"`
	Pro  TypeLimits `json:"pro"`
}

// GetLimit returns the limit for a specific resource type
func (t TypeLimits) GetLimit(resource string) int {
	switch resource {
	case "link_import":
		return t.Link
	case "text_import":
		return t.Text
	case "image_import":
		return t.Image
	case "youtube_import":
		return t.YouTube
	default:
		return 0
	}
}
