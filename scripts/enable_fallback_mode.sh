#!/bin/bash

echo "Setting QCAT fallback mode environment variable..."
export QCAT_FALLBACK_MODE=true
echo "QCAT_FALLBACK_MODE=$QCAT_FALLBACK_MODE"

echo ""
echo "Fallback mode enabled! This will:"
echo "- Generate synthetic market data when Binance API is unavailable"
echo "- Allow optimization tasks to continue even with network issues"
echo "- Use realistic price movements based on the trading symbol"
echo ""

echo "Starting QCAT with fallback mode..."
go run cmd/main.go
