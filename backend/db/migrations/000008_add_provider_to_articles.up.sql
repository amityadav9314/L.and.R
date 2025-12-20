-- Add provider column to daily_articles
ALTER TABLE daily_articles ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'tavily';

-- Update uniqueness or indices if needed
-- We already have idx_daily_articles_user_date, but maybe we want to allow 
-- same URL from different providers? Or just keep it as is.
-- The user wants both to run, so we might have duplicate URLs across providers if they find the same thing.
-- But for now, let's just add the column.
