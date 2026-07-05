-- Add geo-location columns to clicks table
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE clicks ADD COLUMN IF NOT EXISTS city VARCHAR(100);
