package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"time"
)

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Algorithm     string        `json:"algorithm"`      // 加密算法：AES-256-GCM, RSA-2048
	KeySize       int           `json:"key_size"`       // 密钥大小
	KeyRotation   time.Duration `json:"key_rotation"`   // 密钥轮换周期
	MasterKey     string        `json:"master_key"`     // 主密钥
	PublicKeyPath string        `json:"public_key_path"` // 公钥路径
	PrivateKeyPath string       `json:"private_key_path"` // 私钥路径
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled     bool   `json:"enabled"`
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	MinVersion  uint16 `json:"min_version"`
	MaxVersion  uint16 `json:"max_version"`
	CipherSuites []uint16 `json:"cipher_suites"`
}

// NetworkSecurityConfig 网络安全配置
type NetworkSecurityConfig struct {
	TLS                    TLSConfig `json:"tls"`
	RateLimitEnabled       bool      `json:"rate_limit_enabled"`
	RateLimitRequests      int       `json:"rate_limit_requests"`
	RateLimitWindow        time.Duration `json:"rate_limit_window"`
	MaxConnections         int       `json:"max_connections"`
	ConnectionTimeout      time.Duration `json:"connection_timeout"`
	IdleTimeout           time.Duration `json:"idle_timeout"`
	ReadTimeout           time.Duration `json:"read_timeout"`
	WriteTimeout          time.Duration `json:"write_timeout"`
	AllowedOrigins        []string  `json:"allowed_origins"`
	AllowedMethods        []string  `json:"allowed_methods"`
	AllowedHeaders        []string  `json:"allowed_headers"`
	ExposedHeaders        []string  `json:"exposed_headers"`
	AllowCredentials      bool      `json:"allow_credentials"`
	MaxAge               time.Duration `json:"max_age"`
}

// EncryptionService 加密服务
type EncryptionService struct {
	config *EncryptionConfig
	rsaKey *rsa.PrivateKey
}

// NewEncryptionService 创建加密服务
func NewEncryptionService(config *EncryptionConfig) (*EncryptionService, error) {
	service := &EncryptionService{
		config: config,
	}
	
	// 加载RSA密钥对
	if err := service.loadRSAKeys(); err != nil {
		return nil, fmt.Errorf("failed to load RSA keys: %w", err)
	}
	
	return service, nil
}

// EncryptData 加密数据
func (e *EncryptionService) EncryptData(data []byte) (string, error) {
	switch e.config.Algorithm {
	case "AES-256-GCM":
		return e.encryptAES(data)
	case "RSA-2048":
		return e.encryptRSA(data)
	default:
		return "", fmt.Errorf("unsupported encryption algorithm: %s", e.config.Algorithm)
	}
}

// DecryptData 解密数据
func (e *EncryptionService) DecryptData(encryptedData string) ([]byte, error) {
	switch e.config.Algorithm {
	case "AES-256-GCM":
		return e.decryptAES(encryptedData)
	case "RSA-2048":
		return e.decryptRSA(encryptedData)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", e.config.Algorithm)
	}
}

// AES加密
func (e *EncryptionService) encryptAES(data []byte) (string, error) {
	// 创建AES cipher
	block, err := aes.NewCipher([]byte(e.config.MasterKey))
	if err != nil {
		return "", err
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// 创建随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	// 加密
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AES解密
func (e *EncryptionService) decryptAES(encryptedData string) ([]byte, error) {
	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher([]byte(e.config.MasterKey))
	if err != nil {
		return nil, err
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	// 提取nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// RSA加密
func (e *EncryptionService) encryptRSA(data []byte) (string, error) {
	if e.rsaKey == nil {
		return "", fmt.Errorf("RSA key not loaded")
	}
	
	// 使用OAEP填充加密
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &e.rsaKey.PublicKey, data, nil)
	if err != nil {
		return "", err
	}
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// RSA解密
func (e *EncryptionService) decryptRSA(encryptedData string) ([]byte, error) {
	if e.rsaKey == nil {
		return nil, fmt.Errorf("RSA key not loaded")
	}
	
	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}
	
	// 使用OAEP填充解密
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, e.rsaKey, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// 加载RSA密钥对
func (e *EncryptionService) loadRSAKeys() error {
	if e.config.PrivateKeyPath == "" {
		return nil // 如果没有配置私钥路径，跳过RSA密钥加载
	}
	
	// 这里应该从文件加载RSA密钥对
	// 为了演示，我们生成一个新的密钥对
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}
	
	e.rsaKey = privateKey
	return nil
}

// GenerateTLSConfig 生成TLS配置
func GenerateTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if !config.Enabled {
		return nil, nil
	}
	
	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}
	
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   config.MinVersion,
		MaxVersion:   config.MaxVersion,
		CipherSuites: config.CipherSuites,
		// 安全配置
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}
	
	return tlsConfig, nil
}

// ValidateNetworkSecurity 验证网络安全配置
func ValidateNetworkSecurity(config *NetworkSecurityConfig) error {
	if config.TLS.Enabled {
		if config.TLS.CertPath == "" || config.TLS.KeyPath == "" {
			return fmt.Errorf("TLS is enabled but certificate or key path is not specified")
		}
	}
	
	if config.RateLimitEnabled {
		if config.RateLimitRequests <= 0 {
			return fmt.Errorf("rate limit requests must be positive")
		}
		if config.RateLimitWindow <= 0 {
			return fmt.Errorf("rate limit window must be positive")
		}
	}
	
	if config.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}
	
	return nil
}

// GenerateSecureRandomBytes 生成安全随机字节
func GenerateSecureRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, bytes)
	return bytes, err
}

// GenerateSecureRandomString 生成安全随机字符串
func GenerateSecureRandomString(length int) (string, error) {
	bytes, err := GenerateSecureRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// HashData 哈希数据
func HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// VerifyHash 验证哈希
func VerifyHash(data []byte, hash string) bool {
	expectedHash := HashData(data)
	return expectedHash == hash
}

// GenerateKeyPair 生成密钥对
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	
	return privateKey, &privateKey.PublicKey, nil
}

// ExportPublicKey 导出公钥
func ExportPublicKey(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}
	
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	
	return string(publicKeyPEM), nil
}

// ExportPrivateKey 导出私钥
func ExportPrivateKey(privateKey *rsa.PrivateKey) (string, error) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	
	return string(privateKeyPEM), nil
}

// ImportPublicKey 导入公钥
func ImportPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}
	
	return rsaPublicKey, nil
}

// ImportPrivateKey 导入私钥
func ImportPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	
	return privateKey, nil
}
