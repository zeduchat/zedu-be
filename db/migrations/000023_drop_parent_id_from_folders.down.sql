ALTER TABLE folders ADD COLUMN parent_id UUID REFERENCES folders(id);
