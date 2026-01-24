-- Seed default quota limits
INSERT INTO settings (key, value, description) VALUES (
    'quota_limits',
    '{"free": {"link": 3, "text": 10, "image": 5, "youtube": 3}, "pro": {"link": 50, "text": 100000, "image": 100, "youtube": 50}}',
    'Daily quota limits for Free and Pro plans'
) ON CONFLICT (key) DO NOTHING;

-- Seed pro access days
INSERT INTO settings (key, value, description) VALUES (
    'pro_access_days',
    '30',
    'Default number of days for Pro subscription access'
) ON CONFLICT (key) DO NOTHING;

