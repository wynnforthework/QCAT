package templates

// Template 策略模板
type Template struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Category    string                   `json:"category"`
	Parameters  map[string]*Parameter    `json:"parameters"`
	Indicators  []string                 `json:"indicators"`
	Signals     map[string]*SignalConfig `json:"signals"`
	RiskRules   map[string]*RiskRule     `json:"risk_rules"`
	Metadata    map[string]interface{}   `json:"metadata"`
}

// Parameter 策略参数定义
type Parameter struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"` // "int", "float", "string", "bool"
	Default     interface{}   `json:"default"`
	Min         interface{}   `json:"min,omitempty"`
	Max         interface{}   `json:"max,omitempty"`
	Options     []interface{} `json:"options,omitempty"` // 枚举选项
	Description string        `json:"description"`
	Required    bool          `json:"required"`
}

// SignalConfig 信号配置
type SignalConfig struct {
	Type        string                 `json:"type"`      // "entry", "exit", "stop"
	Condition   string                 `json:"condition"` // 信号条件表达式
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
}

// RiskRule 风险规则
type RiskRule struct {
	Type        string                 `json:"type"` // "stop_loss", "take_profit", "position_size"
	Condition   string                 `json:"condition"`
	Parameters  map[string]interface{} `json:"parameters"`
	Action      string                 `json:"action"` // "close", "reduce", "alert"
	Description string                 `json:"description"`
}

// NewTrendFollowingTemplate 创建趋势跟踪策略模板
func NewTrendFollowingTemplate() *Template {
	return &Template{
		Name:        "Trend Following",
		Description: "基于移动平均线和动量指标的趋势跟踪策略",
		Category:    "trend",
		Parameters: map[string]*Parameter{
			"ma_period": {
				Name:        "ma_period",
				Type:        "int",
				Default:     20,
				Min:         5,
				Max:         100,
				Description: "移动平均线周期",
				Required:    true,
			},
			"momentum_period": {
				Name:        "momentum_period",
				Type:        "int",
				Default:     14,
				Min:         5,
				Max:         50,
				Description: "动量指标周期",
				Required:    true,
			},
			"entry_threshold": {
				Name:        "entry_threshold",
				Type:        "float",
				Default:     0.02,
				Min:         0.001,
				Max:         0.1,
				Description: "入场阈值",
				Required:    true,
			},
			"stop_loss": {
				Name:        "stop_loss",
				Type:        "float",
				Default:     0.03,
				Min:         0.01,
				Max:         0.1,
				Description: "止损比例",
				Required:    true,
			},
			"take_profit": {
				Name:        "take_profit",
				Type:        "float",
				Default:     0.06,
				Min:         0.02,
				Max:         0.2,
				Description: "止盈比例",
				Required:    true,
			},
		},
		Indicators: []string{"SMA", "EMA", "RSI", "MACD"},
		Signals: map[string]*SignalConfig{
			"long_entry": {
				Type:        "entry",
				Condition:   "price > sma AND rsi > 50 AND macd > 0",
				Priority:    1,
				Description: "多头入场信号",
			},
			"long_exit": {
				Type:        "exit",
				Condition:   "price < sma OR rsi < 30",
				Priority:    1,
				Description: "多头出场信号",
			},
		},
		RiskRules: map[string]*RiskRule{
			"stop_loss": {
				Type:        "stop_loss",
				Condition:   "unrealized_pnl < -stop_loss",
				Action:      "close",
				Description: "止损规则",
			},
			"take_profit": {
				Type:        "take_profit",
				Condition:   "unrealized_pnl > take_profit",
				Action:      "close",
				Description: "止盈规则",
			},
		},
	}
}

// NewMeanReversionTemplate 创建均值回归策略模板
func NewMeanReversionTemplate() *Template {
	return &Template{
		Name:        "Mean Reversion",
		Description: "基于布林带和RSI的均值回归策略",
		Category:    "mean_reversion",
		Parameters: map[string]*Parameter{
			"bb_period": {
				Name:        "bb_period",
				Type:        "int",
				Default:     20,
				Min:         10,
				Max:         50,
				Description: "布林带周期",
				Required:    true,
			},
			"bb_std": {
				Name:        "bb_std",
				Type:        "float",
				Default:     2.0,
				Min:         1.0,
				Max:         3.0,
				Description: "布林带标准差倍数",
				Required:    true,
			},
			"rsi_period": {
				Name:        "rsi_period",
				Type:        "int",
				Default:     14,
				Min:         5,
				Max:         30,
				Description: "RSI周期",
				Required:    true,
			},
			"rsi_oversold": {
				Name:        "rsi_oversold",
				Type:        "float",
				Default:     30.0,
				Min:         20.0,
				Max:         40.0,
				Description: "RSI超卖阈值",
				Required:    true,
			},
			"rsi_overbought": {
				Name:        "rsi_overbought",
				Type:        "float",
				Default:     70.0,
				Min:         60.0,
				Max:         80.0,
				Description: "RSI超买阈值",
				Required:    true,
			},
		},
		Indicators: []string{"BollingerBands", "RSI"},
		Signals: map[string]*SignalConfig{
			"long_entry": {
				Type:        "entry",
				Condition:   "price < bb_lower AND rsi < rsi_oversold",
				Priority:    1,
				Description: "多头入场信号（超卖反弹）",
			},
			"short_entry": {
				Type:        "entry",
				Condition:   "price > bb_upper AND rsi > rsi_overbought",
				Priority:    1,
				Description: "空头入场信号（超买回调）",
			},
		},
	}
}

// NewGridTradingTemplate 创建网格交易策略模板
func NewGridTradingTemplate() *Template {
	return &Template{
		Name:        "Grid Trading",
		Description: "适用于震荡市场的网格交易策略",
		Category:    "grid",
		Parameters: map[string]*Parameter{
			"grid_size": {
				Name:        "grid_size",
				Type:        "float",
				Default:     0.01,
				Min:         0.005,
				Max:         0.05,
				Description: "网格间距比例",
				Required:    true,
			},
			"grid_levels": {
				Name:        "grid_levels",
				Type:        "int",
				Default:     10,
				Min:         5,
				Max:         20,
				Description: "网格层数",
				Required:    true,
			},
			"base_order_size": {
				Name:        "base_order_size",
				Type:        "float",
				Default:     0.1,
				Min:         0.01,
				Max:         1.0,
				Description: "基础订单大小",
				Required:    true,
			},
		},
		Indicators: []string{"SMA", "ATR"},
	}
}

// NewMomentumBreakoutTemplate 创建动量突破策略模板
func NewMomentumBreakoutTemplate() *Template {
	return &Template{
		Name:        "Momentum Breakout",
		Description: "基于价格突破和成交量确认的动量策略",
		Category:    "momentum",
		Parameters: map[string]*Parameter{
			"lookback_period": {
				Name:        "lookback_period",
				Type:        "int",
				Default:     20,
				Min:         10,
				Max:         50,
				Description: "回看周期",
				Required:    true,
			},
			"breakout_threshold": {
				Name:        "breakout_threshold",
				Type:        "float",
				Default:     0.02,
				Min:         0.01,
				Max:         0.05,
				Description: "突破阈值",
				Required:    true,
			},
			"volume_multiplier": {
				Name:        "volume_multiplier",
				Type:        "float",
				Default:     1.5,
				Min:         1.0,
				Max:         3.0,
				Description: "成交量倍数",
				Required:    true,
			},
		},
		Indicators: []string{"HighestHigh", "LowestLow", "Volume", "VolumeMA"},
	}
}

// NewRangeTradingTemplate 创建区间交易策略模板
func NewRangeTradingTemplate() *Template {
	return &Template{
		Name:        "Range Trading",
		Description: "在明确支撑阻力位之间进行区间交易",
		Category:    "range",
		Parameters: map[string]*Parameter{
			"support_level": {
				Name:        "support_level",
				Type:        "float",
				Default:     0.0,
				Description: "支撑位价格",
				Required:    true,
			},
			"resistance_level": {
				Name:        "resistance_level",
				Type:        "float",
				Default:     0.0,
				Description: "阻力位价格",
				Required:    true,
			},
			"range_buffer": {
				Name:        "range_buffer",
				Type:        "float",
				Default:     0.005,
				Min:         0.001,
				Max:         0.02,
				Description: "区间缓冲比例",
				Required:    true,
			},
		},
		Indicators: []string{"Support", "Resistance", "RSI"},
	}
}

// NewShortTrendTemplate 创建空头趋势策略模板
func NewShortTrendTemplate() *Template {
	return &Template{
		Name:        "Short Trend",
		Description: "专门用于下跌趋势的空头策略",
		Category:    "short",
		Parameters: map[string]*Parameter{
			"ma_period": {
				Name:        "ma_period",
				Type:        "int",
				Default:     20,
				Min:         10,
				Max:         50,
				Description: "移动平均线周期",
				Required:    true,
			},
			"rsi_threshold": {
				Name:        "rsi_threshold",
				Type:        "float",
				Default:     60.0,
				Min:         50.0,
				Max:         70.0,
				Description: "RSI入场阈值",
				Required:    true,
			},
		},
		Indicators: []string{"SMA", "RSI", "MACD"},
	}
}

// GetDefaultTemplate 获取默认策略模板
func GetDefaultTemplate() *Template {
	return NewTrendFollowingTemplate()
}
