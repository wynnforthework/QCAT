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

	// 加载K线数据
	klines, err := l.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load klines: %w", err)
	}
	data.Klines = klines

	// 加载订单簿数据
	orderbooks, err := l.orderbookMgr.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load orderbooks: %w", err)
	}
	// Convert OrderBook to Depth
	for _, ob := range orderbooks {
		data.Orderbooks = append(data.Orderbooks, &orderbook.Depth{
			Symbol:    ob.Symbol,
			Bids:      ob.Bids.GetLevels(10),
			Asks:      ob.Asks.GetLevels(10),
			Timestamp: ob.Timestamp,
		})
	}

	// 加载成交数据
	trades, err := l.tradeManager.GetTradeHistory(ctx, symbol, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to load trades: %w", err)
	}
	data.Trades = trades

	// 加载资金费率数据
	fundingRates, err := l.fundingManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load funding rates: %w", err)
	}
	data.FundingRates = fundingRates

	// 加载指数价格数据
	indexPrices, err := l.indexManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load index prices: %w", err)
	}
	data.IndexPrices = indexPrices

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
	if len(d.Orderbooks) == 0 {
		return fmt.Errorf("no orderbook data")
	}
	if len(d.Trades) == 0 {
		return fmt.Errorf("no trade data")
	}
	if len(d.FundingRates) == 0 {
		return fmt.Errorf("no funding rate data")
	}
	if len(d.IndexPrices) == 0 {
		return fmt.Errorf("no index price data")
	}
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
