package prompts

import (
	_ "embed"
)

//go:embed agent_daily_feed.txt
var AgentDailyFeed string

//go:embed query_optimization.txt
var QueryOptimization string

//go:embed flashcards.txt
var Flashcards string

//go:embed summary.txt
var Summary string

//go:embed ocr.txt
var OCR string

//go:embed tool_get_preferences.txt
var ToolGetPreferencesDesc string

//go:embed tool_search_news.txt
var ToolSearchNewsDesc string

//go:embed tool_store_articles.txt
var ToolStoreArticlesDesc string
