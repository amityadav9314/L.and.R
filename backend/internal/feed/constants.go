package feed

const (
	// Article filtering
	MinRelevanceScore = 0.6
	MaxArticlesPerDay = 10

	// Search configuration
	MaxSearchLoops   = 3
	SearchMaxResults = 10

	// Rate limiting
	DelayBetweenSearches = 5   // seconds
	DelayBetweenUsers    = 120 // seconds (2 minutes)
)
