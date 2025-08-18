package market

import (
	"context"
	"qcat/internal/types"
)

// WSClient represents a WebSocket client
type WSClient struct {
	*types.WSClient
}

// NewWSClient creates a new WebSocket client
func NewWSClient(url string) *WSClient {
	return &WSClient{
		WSClient: types.NewWSClient(url),
	}
}

// Connect establishes a WebSocket connection
func (c *WSClient) Connect(ctx context.Context) error {
	return c.WSClient.Connect(ctx)
}

// Subscribe adds a new subscription
func (c *WSClient) Subscribe(sub types.WSSubscription, handler types.MarketDataHandler) error {
	return c.WSClient.Subscribe(sub, handler)
}

// Unsubscribe removes a subscription
func (c *WSClient) Unsubscribe(symbol string, marketType types.MarketType) error {
	return c.WSClient.Unsubscribe(symbol, marketType)
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	return c.WSClient.Close()
}
