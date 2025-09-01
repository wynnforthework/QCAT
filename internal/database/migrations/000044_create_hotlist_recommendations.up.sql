-- Migration: Create hotlist_recommendations table
-- Version: 000044
-- Description: Creates the missing hotlist_recommendations table for storing hot coin recommendations

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create hotlist_recommendations table
CREATE TABLE IF NOT EXISTS hotlist_recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(20) NOT NULL,
    score DECIMAL(10,6) NOT NULL DEFAULT 0,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high'
    risk_score DECIMAL(10,6) NOT NULL DEFAULT 0,
    price_min DECIMAL(30,10),
    price_max DECIMAL(30,10),
    safe_leverage DECIMAL(10,4) DEFAULT 1.0,
    market_sentiment VARCHAR(20) DEFAULT 'neutral', -- 'bullish', 'bearish', 'neutral'
    sentiment_score DECIMAL(10,6) DEFAULT 0,
    reason TEXT,
    tags TEXT, -- Comma-separated tags
    confidence DECIMAL(5,4) DEFAULT 0.5000, -- 0.0000 to 1.0000
    time_horizon VARCHAR(20) DEFAULT 'short', -- 'short', 'medium', 'long'
    expected_return DECIMAL(10,6) DEFAULT 0,
    max_drawdown DECIMAL(10,6) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours'),
    
    -- Constraints
    CONSTRAINT chk_risk_level CHECK (risk_level IN ('low', 'medium', 'high')),
    CONSTRAINT chk_market_sentiment CHECK (market_sentiment IN ('bullish', 'bearish', 'neutral')),
    CONSTRAINT chk_time_horizon CHECK (time_horizon IN ('short', 'medium', 'long')),
    CONSTRAINT chk_confidence_range CHECK (confidence >= 0.0000 AND confidence <= 1.0000),
    CONSTRAINT chk_price_range CHECK (price_min IS NULL OR price_max IS NULL OR price_min <= price_max),
    
    -- Unique constraint to prevent duplicate recommendations for the same symbol
    UNIQUE(symbol)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_symbol ON hotlist_recommendations(symbol);
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_score ON hotlist_recommendations(score DESC);
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_risk_level ON hotlist_recommendations(risk_level);
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_created_at ON hotlist_recommendations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_expires_at ON hotlist_recommendations(expires_at);
CREATE INDEX IF NOT EXISTS idx_hotlist_recommendations_confidence ON hotlist_recommendations(confidence DESC);

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_hotlist_recommendations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Drop trigger if exists and recreate
DROP TRIGGER IF EXISTS update_hotlist_recommendations_updated_at_trigger ON hotlist_recommendations;
CREATE TRIGGER update_hotlist_recommendations_updated_at_trigger
    BEFORE UPDATE ON hotlist_recommendations
    FOR EACH ROW EXECUTE FUNCTION update_hotlist_recommendations_updated_at();

-- Add table comment for documentation
COMMENT ON TABLE hotlist_recommendations IS 'Stores hot coin recommendations with risk assessment and market sentiment analysis';
COMMENT ON COLUMN hotlist_recommendations.symbol IS 'Trading symbol (e.g., BTCUSDT)';
COMMENT ON COLUMN hotlist_recommendations.score IS 'Overall recommendation score (0-1)';
COMMENT ON COLUMN hotlist_recommendations.risk_level IS 'Risk assessment level: low, medium, high';
COMMENT ON COLUMN hotlist_recommendations.risk_score IS 'Numerical risk score (0-1)';
COMMENT ON COLUMN hotlist_recommendations.price_min IS 'Recommended minimum entry price';
COMMENT ON COLUMN hotlist_recommendations.price_max IS 'Recommended maximum entry price';
COMMENT ON COLUMN hotlist_recommendations.safe_leverage IS 'Recommended safe leverage ratio';
COMMENT ON COLUMN hotlist_recommendations.market_sentiment IS 'Current market sentiment: bullish, bearish, neutral';
COMMENT ON COLUMN hotlist_recommendations.sentiment_score IS 'Numerical sentiment score (-1 to 1)';
COMMENT ON COLUMN hotlist_recommendations.reason IS 'Explanation for the recommendation';
COMMENT ON COLUMN hotlist_recommendations.tags IS 'Comma-separated tags for categorization';
COMMENT ON COLUMN hotlist_recommendations.confidence IS 'Confidence level of the recommendation (0-1)';
COMMENT ON COLUMN hotlist_recommendations.time_horizon IS 'Recommended holding period: short, medium, long';
COMMENT ON COLUMN hotlist_recommendations.expected_return IS 'Expected return percentage';
COMMENT ON COLUMN hotlist_recommendations.max_drawdown IS 'Maximum expected drawdown percentage';
COMMENT ON COLUMN hotlist_recommendations.expires_at IS 'When this recommendation expires';

-- Insert some sample data for testing (only if table is empty)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM hotlist_recommendations LIMIT 1) THEN
        INSERT INTO hotlist_recommendations (
            symbol, score, risk_level, risk_score, price_min, price_max,
            safe_leverage, market_sentiment, sentiment_score, reason,
            tags, confidence, time_horizon, expected_return, max_drawdown
        ) VALUES
        ('BTCUSDT', 0.85, 'medium', 0.45, 44000.00, 46000.00, 2.0, 'bullish', 0.7, 'Strong technical indicators and positive market sentiment', 'crypto,major,trending', 0.8500, 'short', 0.15, 0.08),
        ('ETHUSDT', 0.78, 'medium', 0.50, 2900.00, 3100.00, 2.5, 'bullish', 0.6, 'Following Bitcoin momentum with good fundamentals', 'crypto,defi,trending', 0.7800, 'short', 0.12, 0.10),
        ('BNBUSDT', 0.65, 'low', 0.30, 295.00, 305.00, 3.0, 'neutral', 0.3, 'Stable performance with exchange backing', 'crypto,exchange,stable', 0.6500, 'medium', 0.08, 0.05);
        
        RAISE NOTICE 'Inserted sample hotlist recommendations data';
    END IF;
END $$;

-- Success message
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 000044 completed successfully';
    RAISE NOTICE 'Created hotlist_recommendations table with all required fields and constraints';
    RAISE NOTICE 'Added indexes, triggers, and sample data for testing';
END $$;
