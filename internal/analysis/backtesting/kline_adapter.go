package backtesting

import (
	"context"
	"time"

	"qcat/internal/market/kline"
)

// KlineManagerAdapter 适配器，将kline.Manager适配到BacktestDataManager需要的接口
type KlineManagerAdapter struct {
	manager *kline.Manager
}

// NewKlineManagerAdapter 创建K线管理器适配器
func NewKlineManagerAdapter(manager *kline.Manager) *KlineManagerAdapter {
	return &KlineManagerAdapter{
		manager: manager,
	}
}

// GetHistoryWithBackfill 获取历史数据，如果数据库中没有则自动从API回填
func (a *KlineManagerAdapter) GetHistoryWithBackfill(ctx context.Context, symbol string, interval string, start, end time.Time) ([]KlineData, error) {
	// 转换间隔字符串到kline.Interval
	klineInterval := kline.Interval(interval)
	
	// 调用kline.Manager的自动回填功能
	klines, err := a.manager.GetHistoryWithBackfill(ctx, symbol, klineInterval, start, end)
	if err != nil {
		return nil, err
	}
	
	// 转换为KlineData格式
	var result []KlineData
	for _, k := range klines {
		klineData := KlineData{
			Symbol:    k.Symbol,
			OpenTime:  k.OpenTime,
			CloseTime: k.CloseTime,
			Open:      k.Open,
			High:      k.High,
			Low:       k.Low,
			Close:     k.Close,
			Volume:    k.Volume,
		}
		result = append(result, klineData)
	}
	
	return result, nil
}

// EnsureDataAvailable 确保指定时间范围的数据可用，如果不可用则自动回填
func (a *KlineManagerAdapter) EnsureDataAvailable(ctx context.Context, symbol string, interval string, start, end time.Time) error {
	// 转换间隔字符串到kline.Interval
	klineInterval := kline.Interval(interval)
	
	// 调用kline.Manager的确保数据可用功能
	return a.manager.EnsureDataAvailable(ctx, symbol, klineInterval, start, end)
}
