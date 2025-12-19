-- Add feed preference columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS interest_prompt TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS feed_enabled BOOLEAN DEFAULT false;
