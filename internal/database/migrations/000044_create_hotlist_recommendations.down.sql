-- Migration rollback: Drop hotlist_recommendations table
-- Version: 000044
-- Description: Rollback the creation of hotlist_recommendations table

-- Drop trigger
DROP TRIGGER IF EXISTS update_hotlist_recommendations_updated_at_trigger ON hotlist_recommendations;

-- Drop function
DROP FUNCTION IF EXISTS update_hotlist_recommendations_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_hotlist_recommendations_confidence;
DROP INDEX IF EXISTS idx_hotlist_recommendations_expires_at;
DROP INDEX IF EXISTS idx_hotlist_recommendations_created_at;
DROP INDEX IF EXISTS idx_hotlist_recommendations_risk_level;
DROP INDEX IF EXISTS idx_hotlist_recommendations_score;
DROP INDEX IF EXISTS idx_hotlist_recommendations_symbol;

-- Drop table
DROP TABLE IF EXISTS hotlist_recommendations;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000044 rollback completed successfully';
    RAISE NOTICE 'Dropped hotlist_recommendations table and all related objects';
END $$;
