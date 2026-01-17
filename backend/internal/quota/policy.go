package quota

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
