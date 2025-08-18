package market

import (
	"qcat/internal/types"
)

// Re-export types from the types package for backward compatibility
type OrderBook = types.OrderBook
type Level = types.Level
type Trade = types.Trade
type Kline = types.Kline
type FundingRate = types.FundingRate
type OpenInterest = types.OpenInterest
type IndexPrice = types.IndexPrice
type Ticker = types.Ticker
type MarketDataHandler = types.MarketDataHandler
type MarketType = types.MarketType
type WSSubscription = types.WSSubscription
type WSClient = types.WSClient

// Re-export constants
const (
	MarketTypeSpot    = types.MarketTypeSpot
	MarketTypeFutures = types.MarketTypeFutures
	MarketTypeOptions = types.MarketTypeOptions
)
