-- Create orderbook_snapshots table for storing historical order book data
CREATE TABLE orderbook_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    bids JSONB NOT NULL, -- Array of [price, quantity] pairs
    asks JSONB NOT NULL, -- Array of [price, quantity] pairs
    mid_price DECIMAL(30,10),
    spread DECIMAL(30,10),
    total_bid_volume DECIMAL(30,10),
    total_ask_volume DECIMAL(30,10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, timestamp)
);

-- Create indexes for efficient querying
CREATE INDEX idx_orderbook_snapshots_symbol ON orderbook_snapshots(symbol);
CREATE INDEX idx_orderbook_snapshots_timestamp ON orderbook_snapshots(timestamp);
CREATE INDEX idx_orderbook_snapshots_symbol_timestamp ON orderbook_snapshots(symbol, timestamp);

-- Create a function to automatically calculate mid_price and spread
CREATE OR REPLACE FUNCTION calculate_orderbook_metrics()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate mid price from best bid and ask
    IF jsonb_array_length(NEW.bids) > 0 AND jsonb_array_length(NEW.asks) > 0 THEN
        NEW.mid_price := (
            (NEW.bids->0->0)::DECIMAL + (NEW.asks->0->0)::DECIMAL
        ) / 2;
        
        NEW.spread := (NEW.asks->0->0)::DECIMAL - (NEW.bids->0->0)::DECIMAL;
    END IF;
    
    -- Calculate total volumes
    NEW.total_bid_volume := (
        SELECT COALESCE(SUM((value->1)::DECIMAL), 0)
        FROM jsonb_array_elements(NEW.bids) AS value
    );
    
    NEW.total_ask_volume := (
        SELECT COALESCE(SUM((value->1)::DECIMAL), 0)
        FROM jsonb_array_elements(NEW.asks) AS value
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically calculate metrics
CREATE TRIGGER trigger_calculate_orderbook_metrics
    BEFORE INSERT OR UPDATE ON orderbook_snapshots
    FOR EACH ROW
    EXECUTE FUNCTION calculate_orderbook_metrics();
