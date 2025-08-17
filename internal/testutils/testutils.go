package testutils

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"qcat/internal/cache"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/logger"
)

// TestConfig 测试配置
type TestConfig struct {
	UseRealDB    bool
	UseRealCache bool
	LogLevel     logger.LogLevel
	TempDir      string
}

// DefaultTestConfig 默认测试配置
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		UseRealDB:    false,
		UseRealCache: false,
		LogLevel:     logger.LevelError, // 测试时减少日志输出
		TempDir:      "",
	}
}

// TestSuite 测试套件
type TestSuite struct {
	T        *testing.T
	Config   *TestConfig
	DB       *database.DB
	Cache    cache.Cache
	Logger   logger.Logger
	TempDir  string
	Cleanup  []func()
}

// NewTestSuite 创建测试套件
func NewTestSuite(t *testing.T, config *TestConfig) *TestSuite {
	if config == nil {
		config = DefaultTestConfig()
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "qcat_test_*")
	require.NoError(t, err)

	if config.TempDir == "" {
		config.TempDir = tempDir
	}

	// 初始化日志
	logConfig := logger.Config{
		Level:  config.LogLevel,
		Format: logger.FormatText,
		Output: "stdout",
	}
	testLogger := logger.NewLogger(logConfig)

	suite := &TestSuite{
		T:       t,
		Config:  config,
		Logger:  testLogger,
		TempDir: tempDir,
		Cleanup: []func(){},
	}

	// 设置清理函数
	suite.AddCleanup(func() {
		os.RemoveAll(tempDir)
	})

	// 初始化数据库
	if config.UseRealDB {
		suite.setupRealDB()
	} else {
		suite.setupMockDB()
	}

	// 初始化缓存
	if config.UseRealCache {
		suite.setupRealCache()
	} else {
		suite.setupMockCache()
	}

	return suite
}

// AddCleanup 添加清理函数
func (s *TestSuite) AddCleanup(cleanup func()) {
	s.Cleanup = append(s.Cleanup, cleanup)
}

// TearDown 清理测试环境
func (s *TestSuite) TearDown() {
	for i := len(s.Cleanup) - 1; i >= 0; i-- {
		s.Cleanup[i]()
	}
}

// setupRealDB 设置真实数据库
func (s *TestSuite) setupRealDB() {
	// 这里可以连接到测试数据库
	// 暂时使用内存数据库
	s.setupMockDB()
}

// setupMockDB 设置模拟数据库
func (s *TestSuite) setupMockDB() {
	// 创建内存SQLite数据库用于测试
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(s.T, err)

	s.DB = &database.DB{DB: db}
	s.AddCleanup(func() {
		if s.DB != nil {
			s.DB.Close()
		}
	})
}

// setupRealCache 设置真实缓存
func (s *TestSuite) setupRealCache() {
	// 这里可以连接到测试Redis
	// 暂时使用内存缓存
	s.setupMockCache()
}

// setupMockCache 设置模拟缓存
func (s *TestSuite) setupMockCache() {
	s.Cache = cache.NewMemoryCache(1000)
}

// CreateTempFile 创建临时文件
func (s *TestSuite) CreateTempFile(name, content string) string {
	filePath := filepath.Join(s.TempDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(s.T, err)
	return filePath
}

// CreateTempDir 创建临时目录
func (s *TestSuite) CreateTempDir(name string) string {
	dirPath := filepath.Join(s.TempDir, name)
	err := os.MkdirAll(dirPath, 0755)
	require.NoError(s.T, err)
	return dirPath
}

// HTTPTestHelper HTTP测试助手
type HTTPTestHelper struct {
	Router *gin.Engine
	Suite  *TestSuite
}

// NewHTTPTestHelper 创建HTTP测试助手
func NewHTTPTestHelper(suite *TestSuite) *HTTPTestHelper {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	return &HTTPTestHelper{
		Router: router,
		Suite:  suite,
	}
}

// GET 发送GET请求
func (h *HTTPTestHelper) GET(path string, headers map[string]string) *HTTPResponse {
	return h.Request("GET", path, nil, headers)
}

// POST 发送POST请求
func (h *HTTPTestHelper) POST(path string, body interface{}, headers map[string]string) *HTTPResponse {
	return h.Request("POST", path, body, headers)
}

// PUT 发送PUT请求
func (h *HTTPTestHelper) PUT(path string, body interface{}, headers map[string]string) *HTTPResponse {
	return h.Request("PUT", path, body, headers)
}

// DELETE 发送DELETE请求
func (h *HTTPTestHelper) DELETE(path string, headers map[string]string) *HTTPResponse {
	return h.Request("DELETE", path, nil, headers)
}

// Request 发送HTTP请求
func (h *HTTPTestHelper) Request(method, path string, body interface{}, headers map[string]string) *HTTPResponse {
	var bodyReader io.Reader
	
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(h.Suite.T, err)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	
	// 设置默认头
	req.Header.Set("Content-Type", "application/json")
	
	// 设置自定义头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)

	return &HTTPResponse{
		StatusCode: w.Code,
		Body:       w.Body.Bytes(),
		Headers:    w.Header(),
		suite:      h.Suite,
	}
}

// HTTPResponse HTTP响应
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	suite      *TestSuite
}

// AssertStatus 断言状态码
func (r *HTTPResponse) AssertStatus(expectedStatus int) *HTTPResponse {
	assert.Equal(r.suite.T, expectedStatus, r.StatusCode)
	return r
}

// AssertJSON 断言JSON响应
func (r *HTTPResponse) AssertJSON(expected interface{}) *HTTPResponse {
	var actual interface{}
	err := json.Unmarshal(r.Body, &actual)
	require.NoError(r.suite.T, err)
	assert.Equal(r.suite.T, expected, actual)
	return r
}

// AssertContains 断言响应包含指定内容
func (r *HTTPResponse) AssertContains(substring string) *HTTPResponse {
	assert.Contains(r.suite.T, string(r.Body), substring)
	return r
}

// GetJSON 获取JSON响应
func (r *HTTPResponse) GetJSON(target interface{}) error {
	return json.Unmarshal(r.Body, target)
}

// GetString 获取字符串响应
func (r *HTTPResponse) GetString() string {
	return string(r.Body)
}

// MockData 模拟数据生成器
type MockData struct {
	rand *rand.Rand
}

// NewMockData 创建模拟数据生成器
func NewMockData() *MockData {
	return &MockData{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RandomString 生成随机字符串
func (m *MockData) RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[m.rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomInt 生成随机整数
func (m *MockData) RandomInt(min, max int) int {
	return m.rand.Intn(max-min+1) + min
}

// RandomFloat 生成随机浮点数
func (m *MockData) RandomFloat(min, max float64) float64 {
	return min + m.rand.Float64()*(max-min)
}

// RandomBool 生成随机布尔值
func (m *MockData) RandomBool() bool {
	return m.rand.Intn(2) == 1
}

// RandomChoice 从选项中随机选择
func (m *MockData) RandomChoice(choices []string) string {
	return choices[m.rand.Intn(len(choices))]
}

// GenerateStrategy 生成模拟策略数据
func (m *MockData) GenerateStrategy() map[string]interface{} {
	return map[string]interface{}{
		"id":          m.RandomString(10),
		"name":        "Test Strategy " + m.RandomString(5),
		"type":        m.RandomChoice([]string{"trend", "mean_reversion", "arbitrage"}),
		"status":      m.RandomChoice([]string{"active", "inactive", "testing"}),
		"parameters": map[string]interface{}{
			"ma_short":      m.RandomInt(5, 30),
			"ma_long":       m.RandomInt(30, 100),
			"stop_loss":     m.RandomFloat(0.01, 0.1),
			"take_profit":   m.RandomFloat(0.02, 0.2),
			"leverage":      m.RandomInt(1, 10),
			"position_size": m.RandomFloat(100, 10000),
		},
		"performance": map[string]interface{}{
			"total_return":    m.RandomFloat(-0.5, 2.0),
			"sharpe_ratio":    m.RandomFloat(0.5, 3.0),
			"max_drawdown":    m.RandomFloat(0.05, 0.3),
			"win_rate":        m.RandomFloat(0.3, 0.8),
			"trade_count":     m.RandomInt(10, 1000),
		},
		"created_at": time.Now().Add(-time.Duration(m.RandomInt(1, 365)) * 24 * time.Hour),
		"updated_at": time.Now().Add(-time.Duration(m.RandomInt(0, 7)) * 24 * time.Hour),
	}
}

// GenerateOrder 生成模拟订单数据
func (m *MockData) GenerateOrder() map[string]interface{} {
	return map[string]interface{}{
		"id":         m.RandomString(12),
		"symbol":     m.RandomChoice([]string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT"}),
		"side":       m.RandomChoice([]string{"BUY", "SELL"}),
		"type":       m.RandomChoice([]string{"MARKET", "LIMIT", "STOP"}),
		"quantity":   m.RandomFloat(0.001, 10),
		"price":      m.RandomFloat(100, 50000),
		"status":     m.RandomChoice([]string{"NEW", "FILLED", "CANCELED", "REJECTED"}),
		"created_at": time.Now().Add(-time.Duration(m.RandomInt(0, 24)) * time.Hour),
		"filled_at":  time.Now().Add(-time.Duration(m.RandomInt(0, 23)) * time.Hour),
	}
}

// AssertionHelper 断言助手
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper 创建断言助手
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// AssertNoError 断言无错误
func (a *AssertionHelper) AssertNoError(err error, msgAndArgs ...interface{}) {
	assert.NoError(a.t, err, msgAndArgs...)
}

// AssertError 断言有错误
func (a *AssertionHelper) AssertError(err error, msgAndArgs ...interface{}) {
	assert.Error(a.t, err, msgAndArgs...)
}

// AssertEqual 断言相等
func (a *AssertionHelper) AssertEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.Equal(a.t, expected, actual, msgAndArgs...)
}

// AssertNotEqual 断言不相等
func (a *AssertionHelper) AssertNotEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.NotEqual(a.t, expected, actual, msgAndArgs...)
}

// AssertTrue 断言为真
func (a *AssertionHelper) AssertTrue(value bool, msgAndArgs ...interface{}) {
	assert.True(a.t, value, msgAndArgs...)
}

// AssertFalse 断言为假
func (a *AssertionHelper) AssertFalse(value bool, msgAndArgs ...interface{}) {
	assert.False(a.t, value, msgAndArgs...)
}

// AssertContains 断言包含
func (a *AssertionHelper) AssertContains(s, contains interface{}, msgAndArgs ...interface{}) {
	assert.Contains(a.t, s, contains, msgAndArgs...)
}

// AssertNotContains 断言不包含
func (a *AssertionHelper) AssertNotContains(s, contains interface{}, msgAndArgs ...interface{}) {
	assert.NotContains(a.t, s, contains, msgAndArgs...)
}

// AssertLen 断言长度
func (a *AssertionHelper) AssertLen(object interface{}, length int, msgAndArgs ...interface{}) {
	assert.Len(a.t, object, length, msgAndArgs...)
}

// AssertEmpty 断言为空
func (a *AssertionHelper) AssertEmpty(object interface{}, msgAndArgs ...interface{}) {
	assert.Empty(a.t, object, msgAndArgs...)
}

// AssertNotEmpty 断言不为空
func (a *AssertionHelper) AssertNotEmpty(object interface{}, msgAndArgs ...interface{}) {
	assert.NotEmpty(a.t, object, msgAndArgs...)
}

// AssertNil 断言为nil
func (a *AssertionHelper) AssertNil(object interface{}, msgAndArgs ...interface{}) {
	assert.Nil(a.t, object, msgAndArgs...)
}

// AssertNotNil 断言不为nil
func (a *AssertionHelper) AssertNotNil(object interface{}, msgAndArgs ...interface{}) {
	assert.NotNil(a.t, object, msgAndArgs...)
}

// AssertPanics 断言会panic
func (a *AssertionHelper) AssertPanics(f func(), msgAndArgs ...interface{}) {
	assert.Panics(a.t, f, msgAndArgs...)
}

// AssertNotPanics 断言不会panic
func (a *AssertionHelper) AssertNotPanics(f func(), msgAndArgs ...interface{}) {
	assert.NotPanics(a.t, f, msgAndArgs...)
}

// TimeoutContext 创建带超时的上下文
func TimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WaitForCondition 等待条件满足
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ctx, cancel := TimeoutContext(timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for condition: %s", message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// Eventually 最终断言
func Eventually(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	WaitForCondition(t, condition, timeout, message)
}

// Consistently 持续断言
func Consistently(t *testing.T, condition func() bool, duration time.Duration, message string) {
	end := time.Now().Add(duration)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(end) {
		select {
		case <-ticker.C:
			if !condition() {
				t.Fatalf("Condition failed during consistency check: %s", message)
			}
		}
	}
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DirExists 检查目录是否存在
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// ReadTestFile 读取测试文件
func ReadTestFile(t *testing.T, path string) []byte {
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

// WriteTestFile 写入测试文件
func WriteTestFile(t *testing.T, path string, data []byte) {
	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)
}

// CaptureOutput 捕获输出
func CaptureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// SetEnv 设置环境变量（测试结束后自动恢复）
func SetEnv(t *testing.T, key, value string) {
	oldValue := os.Getenv(key)
	os.Setenv(key, value)
	
	t.Cleanup(func() {
		if oldValue == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, oldValue)
		}
	})
}

// RandomPort 获取随机端口
func RandomPort() int {
	return rand.Intn(10000) + 20000
}

// IsPortAvailable 检查端口是否可用
func IsPortAvailable(port int) bool {
	// 这里可以实现端口检查逻辑
	// 简单起见，直接返回true
	return true
}

// GetAvailablePort 获取可用端口
func GetAvailablePort() int {
	for {
		port := RandomPort()
		if IsPortAvailable(port) {
			return port
		}
	}
}
