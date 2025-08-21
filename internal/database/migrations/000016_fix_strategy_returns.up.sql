-- Migration: Fix strategy returns and performance tables
-- Version: 000016
-- Description: Creates strategy_returns table and adds missing fields to strategy_performance

-- Create strategy_returns table
CREATE TABLE IF NOT EXISTS strategy_returns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID NOT NULL REFERENCES strategies(id),
    return_value DECIMAL(10,6) NOT NULL,
    return_type VARCHAR(20) NOT NULL DEFAULT 'daily',
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add missing fields to strategy_performance table
ALTER TABLE strategy_performance 
ADD COLUMN IF NOT EXISTS daily_return DECIMAL(10,6) DEFAULT 0;

ALTER TABLE strategy_performance 
ADD COLUMN IF NOT EXISTS date DATE;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_strategy_returns_strategy_id ON strategy_returns(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_returns_created_at ON strategy_returns(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_returns_return_type ON strategy_returns(return_type);
CREATE INDEX IF NOT EXISTS idx_strategy_performance_date ON strategy_performance(date DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_performance_daily_return ON strategy_performance(daily_return DESC);

-- Update daily_return field with calculated values from daily_pnl
UPDATE strategy_performance 
SET daily_return = CASE 
    WHEN daily_pnl != 0 AND total_pnl != 0 THEN daily_pnl / NULLIF(total_pnl, 0) * 100
    ELSE 0 
END
WHERE daily_return = 0 AND daily_pnl != 0;

-- Update date field with timestamp date
UPDATE strategy_performance 
SET date = timestamp::date 
WHERE date IS NULL;

-- Add comments for documentation
COMMENT ON TABLE strategy_returns IS 'Stores strategy return values over different time periods';
COMMENT ON COLUMN strategy_returns.return_value IS 'Return value as percentage (e.g., 0.05 for 5%)';
COMMENT ON COLUMN strategy_returns.return_type IS 'Type of return: daily, weekly, monthly';
COMMENT ON COLUMN strategy_performance.daily_return IS 'Daily return percentage calculated from daily_pnl';
COMMENT ON COLUMN strategy_performance.date IS 'Date extracted from timestamp for easier querying';
