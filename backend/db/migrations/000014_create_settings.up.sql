CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed default quota limits
INSERT INTO settings (key, value, description) VALUES (
    'quota_limits',
    '{"free": {"link": 3, "text": 10, "image": 5, "youtube": 3}, "pro": {"link": 50, "text": 100000, "image": 100, "youtube": 50}}',
    'Daily quota limits for Free and Pro plans'
) ON CONFLICT (key) DO NOTHING;
