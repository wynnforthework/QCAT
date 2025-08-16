-- Drop trigger and function
DROP TRIGGER IF EXISTS trigger_calculate_orderbook_metrics ON orderbook_snapshots;
DROP FUNCTION IF EXISTS calculate_orderbook_metrics();

-- Drop table
DROP TABLE IF EXISTS orderbook_snapshots;
