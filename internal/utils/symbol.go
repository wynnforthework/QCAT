package utils

import (
	"log"
	"strings"
)

// NormalizeSymbol 标准化交易对符号，将各种格式转换为交易所API接受的格式
func NormalizeSymbol(symbol string) string {
	// 移除常见的分隔符和后缀
	normalized := strings.ToUpper(symbol)
	original := normalized

	// 处理格式如 "ETH/USDT:USDT" -> "ETHUSDT"
	if strings.Contains(normalized, "/") {
		parts := strings.Split(normalized, "/")
		if len(parts) >= 2 {
			base := strings.TrimSpace(parts[0])
			quote := strings.TrimSpace(parts[1])
			
			// 移除冒号后的部分，如 "USDT:USDT" -> "USDT"
			if strings.Contains(quote, ":") {
				quoteParts := strings.Split(quote, ":")
				quote = strings.TrimSpace(quoteParts[0])
			}
			
			// 验证base和quote不为空
			if base != "" && quote != "" {
				normalized = base + quote
			}
		}
	}

	// 处理格式如 "ETH-USDT" -> "ETHUSDT"
	normalized = strings.ReplaceAll(normalized, "-", "")

	// 处理格式如 "ETH_USDT" -> "ETHUSDT"
	normalized = strings.ReplaceAll(normalized, "_", "")

	// 移除其他特殊字符
	normalized = strings.ReplaceAll(normalized, ":", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, " ", "")

	// 验证结果格式
	if !IsValidSymbolFormat(normalized) {
		log.Printf("Warning: Symbol format may be invalid after normalization: %s -> %s", original, normalized)
		// 尝试修复常见问题
		normalized = FixCommonSymbolIssues(normalized)
	}

	log.Printf("Symbol normalized: %s -> %s", original, normalized)
	return normalized
}

// IsValidSymbolFormat 验证符号格式是否有效
func IsValidSymbolFormat(symbol string) bool {
	// 基本长度检查
	if len(symbol) < 6 || len(symbol) > 20 {
		return false
	}
	
	// 检查是否只包含字母和数字
	for _, r := range symbol {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	
	// 检查是否以常见的计价货币结尾
	commonQuotes := []string{"USDT", "USDC", "BTC", "ETH", "BNB", "BUSD", "DAI", "TUSD"}
	for _, quote := range commonQuotes {
		if strings.HasSuffix(symbol, quote) {
			return true
		}
	}
	
	return false
}

// FixCommonSymbolIssues 修复常见的符号问题
func FixCommonSymbolIssues(symbol string) string {
	// 移除重复的USDT
	if strings.Contains(symbol, "USDTUSDT") {
		symbol = strings.ReplaceAll(symbol, "USDTUSDT", "USDT")
	}
	
	// 移除重复的USDC
	if strings.Contains(symbol, "USDCUSDC") {
		symbol = strings.ReplaceAll(symbol, "USDCUSDC", "USDC")
	}
	
	// 移除重复的BTC
	if strings.Contains(symbol, "BTCBTC") {
		symbol = strings.ReplaceAll(symbol, "BTCBTC", "BTC")
	}
	
	// 移除重复的ETH
	if strings.Contains(symbol, "ETHETH") {
		symbol = strings.ReplaceAll(symbol, "ETHETH", "ETH")
	}
	
	// 移除重复的BNB
	if strings.Contains(symbol, "BNBBNB") {
		symbol = strings.ReplaceAll(symbol, "BNBBNB", "BNB")
	}
	
	return symbol
}

// ParseSymbol 解析交易对符号，返回基础资产和计价资产
func ParseSymbol(symbol string) (baseAsset, quoteAsset string) {
	normalized := NormalizeSymbol(symbol)
	
	// 常见的计价货币，按长度排序（长的在前面）
	commonQuotes := []string{"USDT", "USDC", "BUSD", "TUSD", "BTC", "ETH", "BNB", "DAI"}
	
	for _, quote := range commonQuotes {
		if strings.HasSuffix(normalized, quote) {
			baseAsset = strings.TrimSuffix(normalized, quote)
			quoteAsset = quote
			return
		}
	}
	
	// 如果没有匹配到常见计价货币，默认假设是USDT交易对
	if len(normalized) > 4 {
		baseAsset = normalized[:len(normalized)-4]
		quoteAsset = normalized[len(normalized)-4:]
	} else {
		baseAsset = normalized
		quoteAsset = "USDT"
	}
	
	return
}

// ValidateSymbolForExchange 验证符号是否适用于特定交易所
func ValidateSymbolForExchange(symbol, exchange string) bool {
	normalized := NormalizeSymbol(symbol)
	
	switch strings.ToLower(exchange) {
	case "binance":
		return validateBinanceSymbol(normalized)
	case "okx":
		return validateOKXSymbol(normalized)
	case "bybit":
		return validateBybitSymbol(normalized)
	default:
		return IsValidSymbolFormat(normalized)
	}
}

// validateBinanceSymbol 验证Binance符号格式
func validateBinanceSymbol(symbol string) bool {
	// Binance符号格式：基础资产+计价资产，如BTCUSDT
	if !IsValidSymbolFormat(symbol) {
		return false
	}
	
	// Binance支持的计价货币
	binanceQuotes := []string{"USDT", "USDC", "BTC", "ETH", "BNB", "BUSD"}
	for _, quote := range binanceQuotes {
		if strings.HasSuffix(symbol, quote) {
			return true
		}
	}
	
	return false
}

// validateOKXSymbol 验证OKX符号格式
func validateOKXSymbol(symbol string) bool {
	// OKX符号格式类似Binance
	return validateBinanceSymbol(symbol)
}

// validateBybitSymbol 验证Bybit符号格式
func validateBybitSymbol(symbol string) bool {
	// Bybit符号格式类似Binance
	return validateBinanceSymbol(symbol)
}

// ConvertSymbolFormat 转换符号格式以适应不同的交易所
func ConvertSymbolFormat(symbol, fromExchange, toExchange string) string {
	// 先标准化符号
	normalized := NormalizeSymbol(symbol)
	
	// 根据目标交易所调整格式
	switch strings.ToLower(toExchange) {
	case "binance":
		return normalized // Binance使用标准格式
	case "okx":
		return normalized // OKX使用标准格式
	case "bybit":
		return normalized // Bybit使用标准格式
	default:
		return normalized
	}
}

// GetSymbolVariants 获取符号的各种可能变体
func GetSymbolVariants(symbol string) []string {
	base, quote := ParseSymbol(symbol)
	
	variants := []string{
		base + quote,           // BTCUSDT
		base + "/" + quote,     // BTC/USDT
		base + "-" + quote,     // BTC-USDT
		base + "_" + quote,     // BTC_USDT
		base + ":" + quote,     // BTC:USDT
		strings.ToLower(base + quote), // btcusdt
		strings.ToLower(base + "/" + quote), // btc/usdt
	}
	
	return variants
}
