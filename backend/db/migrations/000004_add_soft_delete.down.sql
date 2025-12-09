DROP INDEX IF EXISTS idx_materials_is_deleted;
ALTER TABLE materials DROP COLUMN IF EXISTS is_deleted;
ALTER TABLE materials DROP COLUMN IF EXISTS deleted_at;
