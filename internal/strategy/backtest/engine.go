package backtest

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy/sdk"
)

// Engine represents the backtesting engine
type Engine struct {
	data     *HistoricalData
	strategy sdk.Strategy
	eventMgr *EventManager
	orderMgr *OrderManager
	posMgr   *PositionManager
	statsMgr *StatsManager
	config   *Config
}

// Config represents engine configuration
type Config struct {
	InitialCapital float64
	MarginMode     exchange.MarginType
	Leverage       float64
	SlippageModel  SlippageModel
	FeeModel       FeeModel
	LatencyModel   LatencyModel
	// 新增：数据源配置字段
	Symbols       []string
    Interval      string
	StartTime     time.Time
	EndTime       time.Time
	DataTypes     []string
	Capital       float64
	DataFeed      DataFeed
	DataFeedType  string
	DataFeedURL   string
	DataFeedToken string
}

// NewEngine creates a new backtesting engine
func NewEngine(data *HistoricalData, strategy sdk.Strategy, config *Config) *Engine {
	eventMgr := NewEventManager()
	orderMgr := NewOrderManager(config.SlippageModel, config.FeeModel, config.LatencyModel)
	posMgr := NewPositionManager(config.InitialCapital, config.MarginMode, config.Leverage)
	statsMgr := NewStatsManager()

	return &Engine{
		data:     data,
		strategy: strategy,
		eventMgr: eventMgr,
		orderMgr: orderMgr,
		posMgr:   posMgr,
		statsMgr: statsMgr,
		config:   config,
	}
}

// Run runs the backtest
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	if err := e.data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid data: %w", err)
	}

	// 初始化策略
    if err := e.strategy.Initialize(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}
    // 注入执行器（可选）
    if ue, ok := e.strategy.(interface{ SetExecution(exec sdk.Execution) }); ok {
        ue.SetExecution(e)
    }

    // 运行回测循环
    currentTime := e.data.Start
    step := time.Minute
    if d, err := intervalToDuration(e.config.Interval); err == nil {
        step = d
    }
	for currentTime.Before(e.data.End) {
		// 处理事件队列
		e.eventMgr.ProcessEvents(currentTime)

		// 获取当前市场数据
		kline := e.data.GetKlineAt(currentTime)
		if kline != nil {
			if err := e.strategy.OnTick(ctx, kline); err != nil {
				return nil, fmt.Errorf("strategy OnTick error: %w", err)
			}
		}

        orderbook := e.data.GetOrderbookAt(currentTime)
        if orderbook != nil {
            // 撮合订单
            trades := e.orderMgr.Match(orderbook)
            for _, tr := range trades {
                // 更新持仓
                if tr.Side == string(exchange.OrderSideBuy) {
                    _ = e.posMgr.OpenPosition(tr)
                } else {
                    // 这里用 ClosePosition 仅针对简化模型；更完整实现需要区分开仓/平仓方向
                    _ = e.posMgr.ClosePosition(tr)
                }
                // 统计交易
                e.statsMgr.AddTrade(tr)
            }
        }

		trades := e.data.GetTradesAt(currentTime)
		for _, t := range trades {
			if err := e.strategy.OnTick(ctx, t); err != nil {
				return nil, fmt.Errorf("strategy OnTick error: %w", err)
			}
		}

		// 处理资金费率
		fundingRate := e.data.GetFundingRateAt(currentTime)
		if fundingRate != nil {
			e.posMgr.ApplyFundingFee(fundingRate)
		}

		// 更新统计数据
		e.statsMgr.Update(currentTime, e.posMgr.GetEquity())

        // 推进时间
        currentTime = currentTime.Add(step)
	}

	// 获取回测结果
	return e.statsMgr.GetResult(), nil
}

// intervalToDuration converts common kline interval strings to duration
func intervalToDuration(interval string) (time.Duration, error) {
    switch interval {
    case "1m":
        return time.Minute, nil
    case "3m":
        return 3 * time.Minute, nil
    case "5m":
        return 5 * time.Minute, nil
    case "15m":
        return 15 * time.Minute, nil
    case "30m":
        return 30 * time.Minute, nil
    case "1h":
        return time.Hour, nil
    case "2h":
        return 2 * time.Hour, nil
    case "4h":
        return 4 * time.Hour, nil
    case "6h":
        return 6 * time.Hour, nil
    case "8h":
        return 8 * time.Hour, nil
    case "12h":
        return 12 * time.Hour, nil
    case "1d":
        return 24 * time.Hour, nil
    case "3d":
        return 72 * time.Hour, nil
    case "1w":
        return 7 * 24 * time.Hour, nil
    case "1M":
        return 30 * 24 * time.Hour, nil
    default:
        return 0, fmt.Errorf("unsupported interval: %s", interval)
    }
}

// Implement sdk.Execution for the backtest engine
func (e *Engine) PlaceOrder(order *exchange.Order) error {
    if order == nil {
        return fmt.Errorf("nil order")
    }
    // enqueue order and immediate match on available book snapshot
    e.orderMgr.PlaceOrder(order)
    // try to match against current orderbook if available
    // Note: matching is also handled in the loop via data; this provides immediate processing if desired.
    return nil
}

// Result represents backtest results
type Result struct {
	Returns          []float64
	Equity           []float64
	Drawdowns        []float64
	Trades           []*exchange.Trade
	PerformanceStats *PerformanceStats
}

// PerformanceStats represents comprehensive performance statistics
type PerformanceStats struct {
	// 基础收益指标
	Returns         []float64 `json:"returns"`          // 收益率序列
	DailyReturns    []float64 `json:"daily_returns"`    // 日收益率
	TotalReturn     float64   `json:"total_return"`     // 总收益率
	AnnualReturn    float64   `json:"annual_return"`    // 年化收益率
	
	// 风险指标
	Volatility      float64   `json:"volatility"`       // 波动率
	SharpeRatio     float64   `json:"sharpe_ratio"`     // 夏普比率
	SortinoRatio    float64   `json:"sortino_ratio"`    // Sortino比率
	CalmarRatio     float64   `json:"calmar_ratio"`     // Calmar比率
	MaxDrawdown     float64   `json:"max_drawdown"`     // 最大回撤
	
	// 分布特征
	Skewness        float64   `json:"skewness"`         // 偏度
	Kurtosis        float64   `json:"kurtosis"`         // 峰度
	VaR95           float64   `json:"var_95"`           // 95% VaR
	VaR99           float64   `json:"var_99"`           // 99% VaR
	
	// 交易统计
	WinRate         float64   `json:"win_rate"`         // 胜率
	ProfitFactor    float64   `json:"profit_factor"`    // 盈利因子
	TradeCount      int       `json:"trade_count"`      // 交易次数
	AvgTradeReturn  float64   `json:"avg_trade_return"` // 平均交易收益
	AvgHoldingTime  time.Duration `json:"avg_holding_time"` // 平均持仓时间
	
	// 时间相关
	StartDate       time.Time `json:"start_date"`       // 开始日期
	EndDate         time.Time `json:"end_date"`         // 结束日期
	TradingDays     int       `json:"trading_days"`     // 交易天数
}

// SlippageModel defines the interface for slippage models
type SlippageModel interface {
	CalculateSlippage(price, quantity float64, side exchange.OrderSide) float64
}

// FeeModel defines the interface for fee models
type FeeModel interface {
	CalculateFee(price, quantity float64) float64
}

// LatencyModel defines the interface for latency models
type LatencyModel interface {
	GetLatency() time.Duration
}
