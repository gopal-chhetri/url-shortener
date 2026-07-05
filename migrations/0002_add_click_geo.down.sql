-- Remove geo-location columns from clicks table
ALTER TABLE clicks DROP COLUMN IF EXISTS city;
ALTER TABLE clicks DROP COLUMN IF EXISTS country;
ALTER TABLE clicks DROP COLUMN IF EXISTS ip_address;
