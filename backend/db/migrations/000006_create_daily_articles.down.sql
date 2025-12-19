-- Drop daily_articles table
DROP INDEX IF EXISTS idx_daily_articles_user_date;
DROP INDEX IF EXISTS idx_daily_articles_suggested_date;
DROP TABLE IF EXISTS daily_articles;
