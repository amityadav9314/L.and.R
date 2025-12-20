ALTER TABLE materials ADD COLUMN source_url TEXT;
CREATE INDEX idx_materials_source_url ON materials(source_url);
