-- Remove feed preference columns from users table
ALTER TABLE users DROP COLUMN IF EXISTS interest_prompt;
ALTER TABLE users DROP COLUMN IF EXISTS feed_enabled;
