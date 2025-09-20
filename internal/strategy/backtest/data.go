package backtest

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/market/funding"
	"qcat/internal/market/index"
	"qcat/internal/market/kline"
	"qcat/internal/market/orderbook"
	"qcat/internal/market/trade"
)

// DataLoader loads historical market data for backtesting
type DataLoader struct {
	klineManager   *kline.Manager
	orderbookMgr   *orderbook.Manager
	tradeManager   *trade.Manager
	fundingManager *funding.Manager
	indexManager   *index.Manager
}

// NewDataLoader creates a new data loader
func NewDataLoader(
	km *kline.Manager,
	om *orderbook.Manager,
	tm *trade.Manager,
	fm *funding.Manager,
	im *index.Manager,
) *DataLoader {
	return &DataLoader{
		klineManager:   km,
		orderbookMgr:   om,
		tradeManager:   tm,
		fundingManager: fm,
		indexManager:   im,
	}
}

// LoadData loads historical data for the specified period
func (l *DataLoader) LoadData(ctx context.Context, symbol string, start, end time.Time) (*HistoricalData, error) {
	data := &HistoricalData{
		Symbol: symbol,
		Start:  start,
		End:    end,
	}

	// 加载K线数据（使用自动回填功能）
	klines, err := l.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load klines: %w", err)
	}
	data.Klines = klines

	// 以下数据若缺失，不再阻断回测，允许仅K线模式
	if l.orderbookMgr != nil {
		if orderbooks, err := l.orderbookMgr.GetHistory(ctx, symbol, start, end); err == nil {
			for _, ob := range orderbooks {
				data.Orderbooks = append(data.Orderbooks, &orderbook.Depth{
					Symbol:    ob.Symbol,
					Bids:      ob.Bids.GetLevels(10),
					Asks:      ob.Asks.GetLevels(10),
					Timestamp: ob.Timestamp,
				})
			}
		}
	}
	if l.tradeManager != nil {
		if trades, err := l.tradeManager.GetTradeHistory(ctx, symbol, 1000); err == nil {
			data.Trades = trades
		}
	}
	if l.fundingManager != nil {
		if fundingRates, err := l.fundingManager.GetHistory(ctx, symbol, start, end); err == nil {
			data.FundingRates = fundingRates
		}
	}
	if l.indexManager != nil {
		if indexPrices, err := l.indexManager.GetHistory(ctx, symbol, start, end); err == nil {
			data.IndexPrices = indexPrices
		}
	}

	return data, nil
}

// HistoricalData represents historical market data
type HistoricalData struct {
	Symbol       string
	Start        time.Time
	End          time.Time
	Klines       []*kline.Kline
	Orderbooks   []*orderbook.Depth
	Trades       []*trade.Trade
	FundingRates []*funding.Rate
	IndexPrices  []*index.Price
}

// Validate validates the loaded data
func (d *HistoricalData) Validate() error {
	if len(d.Klines) == 0 {
		return fmt.Errorf("no kline data")
	}
	// 允许轻量模式，仅使用K线
	return nil
}

// GetKlineAt returns the kline at the specified time
func (d *HistoricalData) GetKlineAt(t time.Time) *kline.Kline {
	for _, k := range d.Klines {
		if k.OpenTime.Equal(t) {
			return k
		}
	}
	return nil
}

// GetOrderbookAt returns the orderbook at the specified time
func (d *HistoricalData) GetOrderbookAt(t time.Time) *orderbook.Depth {
	for _, ob := range d.Orderbooks {
		if ob.Timestamp.Equal(t) {
			return ob
		}
	}
	return nil
}

// GetTradesAt returns trades at the specified time
func (d *HistoricalData) GetTradesAt(t time.Time) []*trade.Trade {
	var trades []*trade.Trade
	for _, tr := range d.Trades {
		if tr.Timestamp.Equal(t) {
			trades = append(trades, tr)
		}
	}
	return trades
}

// GetFundingRateAt returns the funding rate at the specified time
func (d *HistoricalData) GetFundingRateAt(t time.Time) *funding.Rate {
	for _, fr := range d.FundingRates {
		if fr.LastUpdated.Equal(t) {
			return fr
		}
	}
	return nil
}

// GetIndexPriceAt returns the index price at the specified time
func (d *HistoricalData) GetIndexPriceAt(t time.Time) *index.Price {
	for _, ip := range d.IndexPrices {
		if ip.Timestamp.Equal(t) {
			return ip
		}
	}
	return nil
}
