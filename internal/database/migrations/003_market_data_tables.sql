-- Migration: Create market data tables for real data storage
-- Version: 003
-- Description: Creates tables for storing real market data from Binance

-- Market data (klines/candlesticks)
CREATE TABLE IF NOT EXISTS market_data (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    interval VARCHAR(10) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    open DECIMAL(20,8) NOT NULL,
    high DECIMAL(20,8) NOT NULL,
    low DECIMAL(20,8) NOT NULL,
    close DECIMAL(20,8) NOT NULL,
    volume DECIMAL(20,8) NOT NULL DEFAULT 0,
    complete BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(symbol, interval, timestamp)
);

-- Create indexes for market_data
CREATE INDEX IF NOT EXISTS idx_market_data_symbol_interval ON market_data(symbol, interval);
CREATE INDEX IF NOT EXISTS idx_market_data_timestamp ON market_data(timestamp);
CREATE INDEX IF NOT EXISTS idx_market_data_symbol_timestamp ON market_data(symbol, timestamp);

-- Trades table
CREATE TABLE IF NOT EXISTS trades (
    id VARCHAR(50) PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    size DECIMAL(20,8) NOT NULL,
    side VARCHAR(10) NOT NULL CHECK (side IN ('BUY', 'SELL')),
    fee DECIMAL(20,8) DEFAULT 0,
    fee_currency VARCHAR(10) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    INDEX idx_trades_symbol (symbol),
    INDEX idx_trades_created_at (created_at),
    INDEX idx_trades_symbol_created_at (symbol, created_at)
);

-- Order books table
CREATE TABLE IF NOT EXISTS order_books (
    symbol VARCHAR(20) PRIMARY KEY,
    bids JSONB NOT NULL,
    asks JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for order_books
CREATE INDEX IF NOT EXISTS idx_order_books_updated_at ON order_books(updated_at);

-- Funding rates table (for futures)
CREATE TABLE IF NOT EXISTS funding_rates (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    rate DECIMAL(10,8) NOT NULL,
    next_rate DECIMAL(10,8) DEFAULT 0,
    next_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indexes for funding_rates
CREATE INDEX IF NOT EXISTS idx_funding_rates_symbol ON funding_rates(symbol);
CREATE INDEX IF NOT EXISTS idx_funding_rates_created_at ON funding_rates(created_at);
CREATE INDEX IF NOT EXISTS idx_funding_rates_symbol_created_at ON funding_rates(symbol, created_at);

-- Open interest table (for futures)
CREATE TABLE IF NOT EXISTS open_interest (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    value DECIMAL(20,8) NOT NULL,
    notional DECIMAL(20,8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_open_interest_symbol (symbol),
    INDEX idx_open_interest_timestamp (timestamp),
    INDEX idx_open_interest_symbol_timestamp (symbol, timestamp)
);

-- Tickers table (24hr ticker statistics)
CREATE TABLE IF NOT EXISTS tickers (
    symbol VARCHAR(20) PRIMARY KEY,
    price_change DECIMAL(20,8) DEFAULT 0,
    price_change_percent DECIMAL(10,4) DEFAULT 0,
    weighted_avg_price DECIMAL(20,8) DEFAULT 0,
    prev_close_price DECIMAL(20,8) DEFAULT 0,
    last_price DECIMAL(20,8) NOT NULL,
    last_qty DECIMAL(20,8) DEFAULT 0,
    bid_price DECIMAL(20,8) DEFAULT 0,
    bid_qty DECIMAL(20,8) DEFAULT 0,
    ask_price DECIMAL(20,8) DEFAULT 0,
    ask_qty DECIMAL(20,8) DEFAULT 0,
    open_price DECIMAL(20,8) DEFAULT 0,
    high_price DECIMAL(20,8) DEFAULT 0,
    low_price DECIMAL(20,8) DEFAULT 0,
    volume DECIMAL(20,8) DEFAULT 0,
    quote_volume DECIMAL(20,8) DEFAULT 0,
    open_time TIMESTAMP WITH TIME ZONE,
    close_time TIMESTAMP WITH TIME ZONE,
    count BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for tickers
CREATE INDEX IF NOT EXISTS idx_tickers_updated_at ON tickers(updated_at);

-- Data quality metrics table
CREATE TABLE IF NOT EXISTS data_quality_metrics (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    total_messages BIGINT DEFAULT 0,
    valid_messages BIGINT DEFAULT 0,
    invalid_messages BIGINT DEFAULT 0,
    missing_messages BIGINT DEFAULT 0,
    duplicate_messages BIGINT DEFAULT 0,
    latency_p50_ms INTEGER DEFAULT 0,
    latency_p95_ms INTEGER DEFAULT 0,
    latency_p99_ms INTEGER DEFAULT 0,
    quality_score DECIMAL(5,4) DEFAULT 0,
    last_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(symbol, data_type)
);

-- Create indexes for data_quality_metrics
CREATE INDEX IF NOT EXISTS idx_data_quality_symbol ON data_quality_metrics(symbol);
CREATE INDEX IF NOT EXISTS idx_data_quality_last_update ON data_quality_metrics(last_update);

-- Data quality issues table
CREATE TABLE IF NOT EXISTS data_quality_issues (
    id VARCHAR(100) PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    issue_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    description TEXT NOT NULL,
    data JSONB,
    resolved BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for data_quality_issues
CREATE INDEX IF NOT EXISTS idx_data_quality_issues_symbol ON data_quality_issues(symbol);
CREATE INDEX IF NOT EXISTS idx_data_quality_issues_severity ON data_quality_issues(severity);
CREATE INDEX IF NOT EXISTS idx_data_quality_issues_resolved ON data_quality_issues(resolved);
CREATE INDEX IF NOT EXISTS idx_data_quality_issues_created_at ON data_quality_issues(created_at);

-- Market data ingestion statistics table
CREATE TABLE IF NOT EXISTS ingestion_stats (
    id BIGSERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    messages_total BIGINT DEFAULT 0,
    messages_valid BIGINT DEFAULT 0,
    messages_invalid BIGINT DEFAULT 0,
    last_message TIMESTAMP WITH TIME ZONE,
    avg_latency_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(symbol, data_type)
);

-- Create indexes for ingestion_stats
CREATE INDEX IF NOT EXISTS idx_ingestion_stats_symbol ON ingestion_stats(symbol);
CREATE INDEX IF NOT EXISTS idx_ingestion_stats_updated_at ON ingestion_stats(updated_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_market_data_updated_at BEFORE UPDATE ON market_data FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tickers_updated_at BEFORE UPDATE ON tickers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ingestion_stats_updated_at BEFORE UPDATE ON ingestion_stats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE market_data IS 'Stores OHLCV candlestick data from exchanges';
COMMENT ON TABLE trades IS 'Stores individual trade executions';
COMMENT ON TABLE order_books IS 'Stores order book snapshots';
COMMENT ON TABLE funding_rates IS 'Stores funding rates for perpetual futures';
COMMENT ON TABLE open_interest IS 'Stores open interest data for futures';
COMMENT ON TABLE tickers IS 'Stores 24hr ticker statistics';
COMMENT ON TABLE data_quality_metrics IS 'Tracks data quality metrics for monitoring';
COMMENT ON TABLE data_quality_issues IS 'Records data quality issues and anomalies';
COMMENT ON TABLE ingestion_stats IS 'Tracks data ingestion statistics and performance';