-- Fix migrations if tables already exist
-- Run this if you get "relation already exists" errors

-- Create migrations table if not exists
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Mark 001 as applied if videos table exists
INSERT INTO schema_migrations (version) 
SELECT '001' 
WHERE EXISTS (
    SELECT 1 FROM information_schema.tables 
    WHERE table_name = 'videos'
) 
AND NOT EXISTS (
    SELECT 1 FROM schema_migrations WHERE version = '001'
);

-- Show current migration status
SELECT * FROM schema_migrations ORDER BY version;