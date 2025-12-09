ALTER TABLE materials ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE materials ADD COLUMN deleted_at TIMESTAMP;
CREATE INDEX idx_materials_is_deleted ON materials(is_deleted);
