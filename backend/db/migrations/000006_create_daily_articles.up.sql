-- Create daily_articles table
CREATE TABLE IF NOT EXISTS daily_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    snippet TEXT,
    suggested_date DATE NOT NULL,
    relevance_score FLOAT DEFAULT 0.5,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indices for efficient querying
CREATE INDEX IF NOT EXISTS idx_daily_articles_user_date ON daily_articles(user_id, suggested_date);
CREATE INDEX IF NOT EXISTS idx_daily_articles_suggested_date ON daily_articles(suggested_date);
