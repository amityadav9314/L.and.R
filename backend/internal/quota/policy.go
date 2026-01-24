package quota

// Resource types
const (
	ResourceLinkImport    = "link_import"
	ResourceTextImport    = "text_import"
	ResourceImageImport   = "image_import"
	ResourceYoutubeImport = "youtube_import"
	// Future: ResourcePdfImport = "pdf_import"
)

// ResourceDisplayName returns a user-friendly name for error messages
func ResourceDisplayName(resource string) string {
	switch resource {
	case ResourceLinkImport:
		return "link imports"
	case ResourceTextImport:
		return "text imports"
	case ResourceImageImport:
		return "image imports"
	case ResourceYoutubeImport:
		return "YouTube imports"
	default:
		return resource
	}
}

// Limits (defaults, can be overridden by env vars)
const (
	FreeLinkLimit = 3
	FreeTextLimit = 10

	// Effectively unlimited for Pro
	ProLinkLimit = 50
	ProTextLimit = 100000
)
