-- Remove provider column from daily_articles
ALTER TABLE daily_articles DROP COLUMN IF NOT EXISTS provider;
