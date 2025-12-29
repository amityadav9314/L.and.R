package models

const (
	// === Groq Models ===
	ModelGroqLlama3_1_8b     = "llama-3.1-8b-instant"
	ModelGroqLlama3_3_70b    = "llama-3.3-70b-versatile"
	ModelGroqLlamaGuard4_12b = "meta-llama/llama-guard-4-12b"
	ModelGroqGptOss120b      = "openai/gpt-oss-120b"
	ModelGroqGptOss20b       = "openai/gpt-oss-20b"
	ModelGroqWhisper         = "whisper-large-v3"
	ModelGroqWhisperTurbo    = "whisper-large-v3-turbo"
	ModelGroqQwen_32b        = "qwen/qwen3-32b"

	// Keep existing vision model for OCR compatibility (not in user list but required by code)
	ModelGroqVision = "meta-llama/llama-4-scout-17b-16e-instruct"

	// === Cerebras Models ===
	ModelCerebrasGptOss120b   = "gpt-oss-120b"
	ModelCerebrasLlama3_3_70b = "llama-3.3-70b"
	ModelCerebrasLlama3_1_8b  = "llama3.1-8b"
	ModelCerebrasQwen3_235b   = "qwen-3-235b-a22b-instruct-2507"
	ModelCerebrasQwen3_32b    = "qwen-3-32b"
	ModelCerebrasZaiGlm4_6    = "zai-glm-4.6"
)

const (
	// === Task-Specific Default Models ===

	// TaskAgentDailyFeedModel: Complex tool use, reasoning.
	//TaskAgentDailyFeedModel = ModelGroqQwen_32b
	//TaskAgentDailyFeedModel = ModelGroqLlama3_3_70b
	TaskAgentDailyFeedModel = ModelGroqGptOss120b

	// TaskSummaryModel: High context, instruction following.
	TaskSummaryModel = ModelGroqGptOss120b // Upgraded to Llama 3.3 for better quality

	// TaskFlashcardModel: Structured JSON generation.
	TaskFlashcardModel = ModelGroqGptOss120b

	// TaskVisionModel: OCR
	TaskVisionModel = ModelGroqVision
)
