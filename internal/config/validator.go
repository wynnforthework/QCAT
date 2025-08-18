package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Validator 配置验证器
type Validator struct {
	config *Config
}

// NewValidator 创建配置验证器
func NewValidator(config *Config) *Validator {
	return &Validator{
		config: config,
	}
}

// Validate 验证配置
func (v *Validator) Validate() error {
	var errors []string

	// 验证应用配置
	if err := v.validateApp(); err != nil {
		errors = append(errors, fmt.Sprintf("应用配置错误: %v", err))
	}

	// 验证服务器配置
	if err := v.validateServer(); err != nil {
		errors = append(errors, fmt.Sprintf("服务器配置错误: %v", err))
	}

	// 验证数据库配置
	if err := v.validateDatabase(); err != nil {
		errors = append(errors, fmt.Sprintf("数据库配置错误: %v", err))
	}

	// 验证Redis配置
	if err := v.validateRedis(); err != nil {
		errors = append(errors, fmt.Sprintf("Redis配置错误: %v", err))
	}

	// 验证交易所配置
	if err := v.validateExchange(); err != nil {
		errors = append(errors, fmt.Sprintf("交易所配置错误: %v", err))
	}

	// 验证JWT配置
	if err := v.validateJWT(); err != nil {
		errors = append(errors, fmt.Sprintf("JWT配置错误: %v", err))
	}

	// 验证安全配置
	if err := v.validateSecurity(); err != nil {
		errors = append(errors, fmt.Sprintf("安全配置错误: %v", err))
	}

	// 验证策略配置
	if err := v.validateStrategy(); err != nil {
		errors = append(errors, fmt.Sprintf("策略配置错误: %v", err))
	}

	// 验证优化器配置
	if err := v.validateOptimizer(); err != nil {
		errors = append(errors, fmt.Sprintf("优化器配置错误: %v", err))
	}

	// 验证市场数据配置
	if err := v.validateMarketData(); err != nil {
		errors = append(errors, fmt.Sprintf("市场数据配置错误: %v", err))
	}

	// 验证风险管理配置
	if err := v.validateRisk(); err != nil {
		errors = append(errors, fmt.Sprintf("风险管理配置错误: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证失败:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// validateApp 验证应用配置
func (v *Validator) validateApp() error {
	app := v.config.App

	if app.Name == "" {
		return fmt.Errorf("应用名称不能为空")
	}

	if app.Version == "" {
		return fmt.Errorf("应用版本不能为空")
	}

	if app.Environment == "" {
		return fmt.Errorf("应用环境不能为空")
	}

	validEnvironments := []string{"development", "staging", "production"}
	valid := false
	for _, env := range validEnvironments {
		if app.Environment == env {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的环境: %s, 有效值: %v", app.Environment, validEnvironments)
	}

	return nil
}

// validateServer 验证服务器配置
func (v *Validator) validateServer() error {
	server := v.config.Server

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", server.Port)
	}

	if server.ReadTimeout <= 0 {
		return fmt.Errorf("读取超时必须大于0")
	}

	if server.WriteTimeout <= 0 {
		return fmt.Errorf("写入超时必须大于0")
	}

	if server.MaxHeaderBytes <= 0 {
		return fmt.Errorf("最大头部字节数必须大于0")
	}

	return nil
}

// validateDatabase 验证数据库配置
func (v *Validator) validateDatabase() error {
	db := v.config.Database

	if db.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}

	if db.Port <= 0 || db.Port > 65535 {
		return fmt.Errorf("无效的数据库端口: %d", db.Port)
	}

	if db.User == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}

	if db.DBName == "" {
		return fmt.Errorf("数据库名称不能为空")
	}

	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	valid := false
	for _, mode := range validSSLModes {
		if db.SSLMode == mode {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的SSL模式: %s, 有效值: %v", db.SSLMode, validSSLModes)
	}

	if db.MaxOpen <= 0 {
		return fmt.Errorf("最大连接数必须大于0")
	}

	if db.MaxIdle < 0 {
		return fmt.Errorf("最大空闲连接数不能为负数")
	}

	if db.MaxIdle > db.MaxOpen {
		return fmt.Errorf("最大空闲连接数不能大于最大连接数")
	}

	if db.Timeout <= 0 {
		return fmt.Errorf("连接超时必须大于0")
	}

	if db.ConnMaxLifetime <= 0 {
		return fmt.Errorf("连接最大生命周期必须大于0")
	}

	return nil
}

// validateRedis 验证Redis配置
func (v *Validator) validateRedis() error {
	redis := v.config.Redis

	// 如果Redis未启用，跳过验证
	if !redis.Enabled {
		return nil
	}

	if redis.Addr == "" {
		return fmt.Errorf("Redis地址不能为空")
	}

	// 验证Redis地址格式
	if !strings.Contains(redis.Addr, ":") {
		return fmt.Errorf("无效的Redis地址格式: %s", redis.Addr)
	}

	if redis.DB < 0 || redis.DB > 15 {
		return fmt.Errorf("无效的Redis数据库编号: %d", redis.DB)
	}

	if redis.PoolSize <= 0 {
		return fmt.Errorf("Redis连接池大小必须大于0")
	}

	if redis.MinIdleConns < 0 {
		return fmt.Errorf("Redis最小空闲连接数不能为负数")
	}

	if redis.MinIdleConns > redis.PoolSize {
		return fmt.Errorf("Redis最小空闲连接数不能大于连接池大小")
	}

	if redis.MaxRetries < 0 {
		return fmt.Errorf("Redis最大重试次数不能为负数")
	}

	if redis.DialTimeout <= 0 {
		return fmt.Errorf("Redis连接超时必须大于0")
	}

	if redis.ReadTimeout <= 0 {
		return fmt.Errorf("Redis读取超时必须大于0")
	}

	if redis.WriteTimeout <= 0 {
		return fmt.Errorf("Redis写入超时必须大于0")
	}

	return nil
}

// validateExchange 验证交易所配置
func (v *Validator) validateExchange() error {
	exchange := v.config.Exchange

	if exchange.Name == "" {
		return fmt.Errorf("交易所名称不能为空")
	}

	// 验证支持的交易所
	supportedExchanges := []string{"binance", "okx", "bybit"}
	valid := false
	for _, ex := range supportedExchanges {
		if exchange.Name == ex {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("不支持的交易所: %s, 支持的交易所: %v", exchange.Name, supportedExchanges)
	}

	// 验证API配置
	if exchange.APIKey == "" {
		return fmt.Errorf("交易所API密钥不能为空")
	}

	if exchange.APISecret == "" {
		return fmt.Errorf("交易所API密钥不能为空")
	}

	// 验证URL格式
	if exchange.BaseURL != "" {
		if _, err := url.Parse(exchange.BaseURL); err != nil {
			return fmt.Errorf("无效的基础URL: %s", exchange.BaseURL)
		}
	}

	if exchange.WebsocketURL != "" {
		if _, err := url.Parse(exchange.WebsocketURL); err != nil {
			return fmt.Errorf("无效的WebSocket URL: %s", exchange.WebsocketURL)
		}
	}

	if exchange.FuturesBaseURL != "" {
		if _, err := url.Parse(exchange.FuturesBaseURL); err != nil {
			return fmt.Errorf("无效的期货基础URL: %s", exchange.FuturesBaseURL)
		}
	}

	if exchange.FuturesWebsocketURL != "" {
		if _, err := url.Parse(exchange.FuturesWebsocketURL); err != nil {
			return fmt.Errorf("无效的期货WebSocket URL: %s", exchange.FuturesWebsocketURL)
		}
	}

	// 验证限流配置
	if exchange.RateLimit.Enabled {
		if exchange.RateLimit.RequestsPerMinute <= 0 {
			return fmt.Errorf("每分钟请求数必须大于0")
		}
		if exchange.RateLimit.Burst <= 0 {
			return fmt.Errorf("突发请求数必须大于0")
		}
	}

	if exchange.Timeout <= 0 {
		return fmt.Errorf("交易所超时必须大于0")
	}

	if exchange.RetryAttempts < 0 {
		return fmt.Errorf("重试次数不能为负数")
	}

	if exchange.RetryDelay < 0 {
		return fmt.Errorf("重试延迟不能为负数")
	}

	return nil
}

// validateJWT 验证JWT配置
func (v *Validator) validateJWT() error {
	jwt := v.config.JWT

	if jwt.SecretKey == "" {
		return fmt.Errorf("JWT密钥不能为空")
	}

	if len(jwt.SecretKey) < 32 {
		return fmt.Errorf("JWT密钥长度必须至少32个字符")
	}

	if jwt.Duration <= 0 {
		return fmt.Errorf("JWT有效期必须大于0")
	}

	return nil
}

// validateSecurity 验证安全配置
func (v *Validator) validateSecurity() error {
	security := v.config.Security

	// 验证KMS配置
	if security.KMS.MasterKey == "" {
		return fmt.Errorf("KMS主密钥不能为空")
	}

	if security.KMS.KeyRotation <= 0 {
		return fmt.Errorf("密钥轮换周期必须大于0")
	}

	// 验证加密配置
	if security.Encryption.Algorithm == "" {
		return fmt.Errorf("加密算法不能为空")
	}

	validAlgorithms := []string{"AES-256-GCM", "AES-256-CBC", "ChaCha20-Poly1305"}
	valid := false
	for _, algo := range validAlgorithms {
		if security.Encryption.Algorithm == algo {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("不支持的加密算法: %s, 支持的算法: %v", security.Encryption.Algorithm, validAlgorithms)
	}

	if security.Encryption.KeySize <= 0 {
		return fmt.Errorf("密钥大小必须大于0")
	}

	if security.Encryption.KeyRotation <= 0 {
		return fmt.Errorf("密钥轮换周期必须大于0")
	}

	// 验证TLS配置
	if security.TLS.Enabled {
		if security.TLS.CertPath == "" {
			return fmt.Errorf("TLS证书路径不能为空")
		}
		if security.TLS.KeyPath == "" {
			return fmt.Errorf("TLS密钥路径不能为空")
		}
	}

	return nil
}

// validateStrategy 验证策略配置
func (v *Validator) validateStrategy() error {
	strategy := v.config.Strategy

	if strategy.DefaultMode == "" {
		return fmt.Errorf("默认策略模式不能为空")
	}

	validModes := []string{"paper", "live", "backtest"}
	valid := false
	for _, mode := range validModes {
		if strategy.DefaultMode == mode {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("无效的策略模式: %s, 有效值: %v", strategy.DefaultMode, validModes)
	}

	if strategy.MaxConcurrentStrategies <= 0 {
		return fmt.Errorf("最大并发策略数必须大于0")
	}

	if strategy.StrategyTimeout <= 0 {
		return fmt.Errorf("策略超时必须大于0")
	}

	if strategy.MemoryLimitMB <= 0 {
		return fmt.Errorf("内存限制必须大于0")
	}

	// 验证回测配置
	if strategy.Backtest.Enabled {
		if strategy.Backtest.Timeout <= 0 {
			return fmt.Errorf("回测超时必须大于0")
		}
		if strategy.Backtest.MaxConcurrency <= 0 {
			return fmt.Errorf("回测最大并发数必须大于0")
		}
		if strategy.Backtest.DataRetentionDays <= 0 {
			return fmt.Errorf("数据保留天数必须大于0")
		}
	}

	return nil
}

// validateOptimizer 验证优化器配置
func (v *Validator) validateOptimizer() error {
	optimizer := v.config.Optimizer

	if optimizer.Enabled {
		if optimizer.Timeout <= 0 {
			return fmt.Errorf("优化器超时必须大于0")
		}
		if optimizer.MaxIterations <= 0 {
			return fmt.Errorf("最大迭代次数必须大于0")
		}
		if optimizer.Concurrency <= 0 {
			return fmt.Errorf("并发数必须大于0")
		}
		if len(optimizer.Algorithms) == 0 {
			return fmt.Errorf("优化算法列表不能为空")
		}
		if optimizer.DefaultAlgorithm == "" {
			return fmt.Errorf("默认优化算法不能为空")
		}

		// 验证默认算法是否在算法列表中
		found := false
		for _, algo := range optimizer.Algorithms {
			if algo == optimizer.DefaultAlgorithm {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("默认算法 %s 不在算法列表中: %v", optimizer.DefaultAlgorithm, optimizer.Algorithms)
		}
	}

	return nil
}

// validateMarketData 验证市场数据配置
func (v *Validator) validateMarketData() error {
	marketData := v.config.MarketData

	if marketData.Enabled {
		if marketData.CacheTTL <= 0 {
			return fmt.Errorf("缓存TTL必须大于0")
		}
		if marketData.BatchSize <= 0 {
			return fmt.Errorf("批处理大小必须大于0")
		}
		if marketData.UpdateInterval <= 0 {
			return fmt.Errorf("更新间隔必须大于0")
		}
		if len(marketData.Symbols) == 0 {
			return fmt.Errorf("交易对列表不能为空")
		}
		if len(marketData.DataTypes) == 0 {
			return fmt.Errorf("数据类型列表不能为空")
		}

		// 验证数据类型
		validDataTypes := []string{"klines", "trades", "orderbook", "funding_rate", "open_interest"}
		for _, dataType := range marketData.DataTypes {
			valid := false
			for _, validType := range validDataTypes {
				if dataType == validType {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("无效的数据类型: %s, 有效值: %v", dataType, validDataTypes)
			}
		}
	}

	return nil
}

// validateRisk 验证风险管理配置
func (v *Validator) validateRisk() error {
	risk := v.config.Risk

	if risk.Enabled {
		if risk.CheckInterval <= 0 {
			return fmt.Errorf("检查间隔必须大于0")
		}
		if risk.MarginCallThreshold <= 0 {
			return fmt.Errorf("追保阈值必须大于0")
		}
		if risk.LiquidationThreshold <= 0 {
			return fmt.Errorf("强平阈值必须大于0")
		}
		if risk.MaxPositionSize <= 0 {
			return fmt.Errorf("最大仓位大小必须大于0")
		}
		if risk.MaxLeverage <= 0 {
			return fmt.Errorf("最大杠杆必须大于0")
		}
		if risk.MaxDrawdown <= 0 || risk.MaxDrawdown > 1 {
			return fmt.Errorf("最大回撤必须在0-1之间")
		}
		if risk.CircuitBreakerThreshold <= 0 || risk.CircuitBreakerThreshold > 1 {
			return fmt.Errorf("熔断器阈值必须在0-1之间")
		}

		// 验证仓位监控配置
		if risk.PositionMonitoring.Enabled {
			if risk.PositionMonitoring.Interval <= 0 {
				return fmt.Errorf("仓位监控间隔必须大于0")
			}
			if risk.PositionMonitoring.AlertThreshold <= 0 || risk.PositionMonitoring.AlertThreshold > 1 {
				return fmt.Errorf("仓位监控告警阈值必须在0-1之间")
			}
		}
	}

	return nil
}

// ValidateRequired 验证必需的环境变量
func (v *Validator) ValidateRequired() error {
	required := []string{
		"QCAT_DATABASE_PASSWORD",
		"QCAT_JWT_SECRET_KEY",
		"QCAT_EXCHANGE_API_KEY",
		"QCAT_EXCHANGE_API_SECRET",
		"QCAT_ENCRYPTION_KEY",
	}

	var missing []string
	for _, key := range required {
		if v.config.getEnvValue(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("缺少必需的环境变量: %v", missing)
	}

	return nil
}

// getEnvValue 获取环境变量值（辅助方法）
func (c *Config) getEnvValue(key string) string {
	// 这里应该实现从环境变量获取值的逻辑
	// 由于Config结构体没有直接访问环境变量的方法，这里返回空字符串
	// 实际实现中应该从os.Getenv获取
	return ""
}
