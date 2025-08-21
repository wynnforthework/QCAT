-- Migration: Add missing strategy fields
-- Version: 000019
-- Description: Adds missing fields for strategy management

-- Add missing fields to strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS is_running BOOLEAN DEFAULT false;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS parameters JSONB DEFAULT '{}';
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS risk_level INTEGER DEFAULT 1;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS max_position_size DECIMAL(15,8) DEFAULT 1000;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS stop_loss_pct DECIMAL(5,4) DEFAULT 0.05;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS take_profit_pct DECIMAL(5,4) DEFAULT 0.10;

-- Update existing strategies to be enabled and set default parameters
UPDATE strategies SET 
    enabled = true,
    is_running = false,
    parameters = CASE 
        WHEN name LIKE '%动量%' THEN '{"period": 14, "threshold": 0.02, "volume_factor": 1.5}'::jsonb
        WHEN name LIKE '%均值回归%' THEN '{"period": 20, "deviation": 2.0, "reversion_threshold": 0.03}'::jsonb
        ELSE '{"period": 10, "threshold": 0.01}'::jsonb
    END,
    risk_level = 2,
    max_position_size = 5000,
    stop_loss_pct = 0.03,
    take_profit_pct = 0.08
WHERE parameters IS NULL OR parameters = '{}';

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_strategies_is_running ON strategies(is_running);
CREATE INDEX IF NOT EXISTS idx_strategies_enabled ON strategies(enabled);
CREATE INDEX IF NOT EXISTS idx_strategies_status_enabled ON strategies(status, enabled);
CREATE INDEX IF NOT EXISTS idx_strategies_risk_level ON strategies(risk_level);

-- Add comments for documentation
COMMENT ON COLUMN strategies.is_running IS 'Whether the strategy is currently running';
COMMENT ON COLUMN strategies.enabled IS 'Whether the strategy is enabled for trading';
COMMENT ON COLUMN strategies.parameters IS 'Strategy-specific parameters in JSON format';
COMMENT ON COLUMN strategies.risk_level IS 'Risk level from 1 (low) to 5 (high)';
COMMENT ON COLUMN strategies.max_position_size IS 'Maximum position size for this strategy';
COMMENT ON COLUMN strategies.stop_loss_pct IS 'Stop loss percentage (0.05 = 5%)';
COMMENT ON COLUMN strategies.take_profit_pct IS 'Take profit percentage (0.10 = 10%)';
