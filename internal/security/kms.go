package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// KMSConfig 密钥管理服务配置
type KMSConfig struct {
	MasterKey     string        // 主密钥
	KeyRotation   time.Duration // 密钥轮换周期
	EncryptionKey string        // 加密密钥
}

// KMS 密钥管理服务
type KMS struct {
	config *KMSConfig
	keys   map[string]*APIKey
}

// APIKey API密钥信息
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Exchange    string    `json:"exchange"`
	Key         string    `json:"-"` // 加密存储
	Secret      string    `json:"-"` // 加密存储
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsed    time.Time `json:"last_used"`
	IsActive    bool      `json:"is_active"`
}

// NewKMS 创建密钥管理服务
func NewKMS(config *KMSConfig) *KMS {
	return &KMS{
		config: config,
		keys:   make(map[string]*APIKey),
	}
}

// CreateAPIKey 创建新的API密钥
func (k *KMS) CreateAPIKey(ctx context.Context, name, exchange, key, secret string, permissions []string) (*APIKey, error) {
	// 生成唯一ID
	id := generateKeyID()
	
	// 加密敏感信息
	encryptedKey, err := k.encrypt(key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}
	
	encryptedSecret, err := k.encrypt(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API secret: %w", err)
	}
	
	apiKey := &APIKey{
		ID:          id,
		Name:        name,
		Exchange:    exchange,
		Key:         encryptedKey,
		Secret:      encryptedSecret,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().AddDate(1, 0, 0), // 1年有效期
		IsActive:    true,
	}
	
	k.keys[id] = apiKey
	return apiKey, nil
}

// GetAPIKey 获取API密钥
func (k *KMS) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	apiKey, exists := k.keys[id]
	if !exists {
		return nil, fmt.Errorf("API key not found: %s", id)
	}
	
	if !apiKey.IsActive {
		return nil, fmt.Errorf("API key is inactive: %s", id)
	}
	
	if time.Now().After(apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired: %s", id)
	}
	
	// 更新最后使用时间
	apiKey.LastUsed = time.Now()
	
	return apiKey, nil
}

// GetDecryptedAPIKey 获取解密后的API密钥
func (k *KMS) GetDecryptedAPIKey(ctx context.Context, id string) (string, string, error) {
	apiKey, err := k.GetAPIKey(ctx, id)
	if err != nil {
		return "", "", err
	}
	
	// 解密密钥
	decryptedKey, err := k.decrypt(apiKey.Key)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API key: %w", err)
	}
	
	decryptedSecret, err := k.decrypt(apiKey.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API secret: %w", err)
	}
	
	return decryptedKey, decryptedSecret, nil
}

// UpdateAPIKey 更新API密钥
func (k *KMS) UpdateAPIKey(ctx context.Context, id string, updates map[string]interface{}) error {
	apiKey, exists := k.keys[id]
	if !exists {
		return fmt.Errorf("API key not found: %s", id)
	}
	
	// 更新字段
	if name, ok := updates["name"].(string); ok {
		apiKey.Name = name
	}
	
	if permissions, ok := updates["permissions"].([]string); ok {
		apiKey.Permissions = permissions
	}
	
	if isActive, ok := updates["is_active"].(bool); ok {
		apiKey.IsActive = isActive
	}
	
	return nil
}

// DeleteAPIKey 删除API密钥
func (k *KMS) DeleteAPIKey(ctx context.Context, id string) error {
	if _, exists := k.keys[id]; !exists {
		return fmt.Errorf("API key not found: %s", id)
	}
	
	delete(k.keys, id)
	return nil
}

// ListAPIKeys 列出所有API密钥
func (k *KMS) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	keys := make([]*APIKey, 0, len(k.keys))
	for _, key := range k.keys {
		// 不返回加密的敏感信息
		safeKey := &APIKey{
			ID:          key.ID,
			Name:        key.Name,
			Exchange:    key.Exchange,
			Permissions: key.Permissions,
			CreatedAt:   key.CreatedAt,
			ExpiresAt:   key.ExpiresAt,
			LastUsed:    key.LastUsed,
			IsActive:    key.IsActive,
		}
		keys = append(keys, safeKey)
	}
	
	return keys, nil
}

// RotateKeys 轮换密钥
func (k *KMS) RotateKeys(ctx context.Context) error {
	// 生成新的主密钥
	newMasterKey := generateRandomKey(32)
	
	// 重新加密所有密钥
	for _, apiKey := range k.keys {
		// 解密当前密钥
		decryptedKey, err := k.decrypt(apiKey.Key)
		if err != nil {
			return fmt.Errorf("failed to decrypt key %s: %w", apiKey.ID, err)
		}
		
		decryptedSecret, err := k.decrypt(apiKey.Secret)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret %s: %w", apiKey.ID, err)
		}
		
		// 使用新密钥重新加密
		oldMasterKey := k.config.MasterKey
		k.config.MasterKey = newMasterKey
		
		newEncryptedKey, err := k.encrypt(decryptedKey)
		if err != nil {
			k.config.MasterKey = oldMasterKey
			return fmt.Errorf("failed to re-encrypt key %s: %w", apiKey.ID, err)
		}
		
		newEncryptedSecret, err := k.encrypt(decryptedSecret)
		if err != nil {
			k.config.MasterKey = oldMasterKey
			return fmt.Errorf("failed to re-encrypt secret %s: %w", apiKey.ID, err)
		}
		
		apiKey.Key = newEncryptedKey
		apiKey.Secret = newEncryptedSecret
	}
	
	return nil
}

// 加密数据
func (k *KMS) encrypt(data string) (string, error) {
	plaintext := []byte(data)
	
	// 创建AES cipher
	block, err := aes.NewCipher([]byte(k.config.MasterKey))
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
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	// 返回base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// 解密数据
func (k *KMS) decrypt(encryptedData string) (string, error) {
	// 解码base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}
	
	// 创建AES cipher
	block, err := aes.NewCipher([]byte(k.config.MasterKey))
	if err != nil {
		return "", err
	}
	
	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// 提取nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	
	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// 生成随机密钥
func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
