DROP INDEX IF EXISTS idx_materials_source_url;
ALTER TABLE materials DROP COLUMN IF EXISTS source_url;
