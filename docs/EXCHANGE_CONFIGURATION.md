# Exchange Configuration Guide

This guide explains how to configure exchanges in QCAT, including how to handle common warnings and issues.

## Basic Exchange Configuration

```go
exchangeConfig := &exchange.ExchangeConfig{
    Name:      "binance",
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",
    TestNet:   true, // Use testnet for development
}
```

## Handling Banexg Library Warnings

When using the banexg library integration, you may see warnings about caching private API results:

```
WARN banexg@v0.2.33-beta.4/biz.go:1170 cache private api result is not recommend
```

### Suppressing Cache Warnings

To attempt suppressing these warnings, enable the `SuppressCacheWarnings` option:

```go
exchangeConfig := &exchange.ExchangeConfig{
    Name:                  "binance",
    APIKey:                "your-api-key",
    APISecret:             "your-api-secret",
    TestNet:               true,
    SuppressCacheWarnings: true, // Attempt to suppress cache warnings
}
```

### What This Does

When `SuppressCacheWarnings` is enabled, the system will:

1. **Disable caching options** in the banexg library configuration
2. **Reduce verbose logging** to minimize log noise
3. **Apply various cache-disabling settings** to the underlying library

### Important Notes

- **Warnings may still appear**: Due to the internal design of the banexg library, some warnings may still occur
- **Functionality is not affected**: These warnings are informational and don't impact trading functionality
- **Safe to ignore**: The warnings are about security best practices, not functional errors

## Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `Name` | string | Exchange name (e.g., "binance") | Required |
| `APIKey` | string | Exchange API key | Required |
| `APISecret` | string | Exchange API secret | Required |
| `TestNet` | bool | Use testnet environment | false |
| `BaseURL` | string | Custom base URL | Auto-detected |
| `FuturesBaseURL` | string | Custom futures base URL | Auto-detected |
| `SuppressCacheWarnings` | bool | Attempt to suppress cache warnings | false |

## Environment Variables

You can also configure exchanges using environment variables:

```bash
export EXCHANGE_API_KEY="your-api-key"
export EXCHANGE_API_SECRET="your-api-secret"
export EXCHANGE_TEST_NET="true"
```

## Best Practices

1. **Always use testnet for development** to avoid accidental real trades
2. **Enable cache warning suppression** if the warnings are causing log noise
3. **Monitor actual functionality** rather than focusing on informational warnings
4. **Keep API credentials secure** and rotate them regularly

## Troubleshooting

### Cache Warnings Still Appear

If cache warnings still appear after enabling `SuppressCacheWarnings`:

1. **This is normal** - the banexg library may have internal caching that cannot be disabled
2. **Check functionality** - ensure your trading operations work correctly
3. **Filter logs** - configure your logging system to filter these specific warnings
4. **Ignore safely** - these warnings don't indicate functional problems

### Connection Issues

If you experience connection issues:

1. **Check API credentials** - ensure they're valid and have the right permissions
2. **Verify network connectivity** - test connection to the exchange
3. **Check rate limits** - ensure you're not exceeding API rate limits
4. **Review testnet settings** - make sure testnet configuration is correct

For more detailed information about banexg warnings, see [BANEXG_WARNINGS.md](BANEXG_WARNINGS.md).
