package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// CryptoTool 加密工具
type CryptoTool struct {
	secretKey []byte
}

// NewCryptoTool 创建加密工具
func NewCryptoTool(secretKey string) *CryptoTool {
	return &CryptoTool{
		secretKey: []byte(secretKey),
	}
}

// HashPassword 使用 Bcrypt 模拟（简化版）
func (c *CryptoTool) HashPassword(password string) (string, error) {
	// 使用 HMAC-SHA256 作为简化实现
	// 实际应使用 golang.org/x/crypto/bcrypt
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyPassword 验证密码
func (c *CryptoTool) VerifyPassword(password, hash string) bool {
	expectedHash, err := c.HashPassword(password)
	if err != nil {
		return false
	}
	return expectedHash == hash
}

// EncryptAES AES 加密
func (c *CryptoTool) EncryptAES(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.secretKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES AES 解密
func (c *CryptoTool) DecryptAES(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.secretKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptString 加密字符串
func (c *CryptoTool) EncryptString(plaintext string) (string, error) {
	encrypted, err := c.EncryptAES([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString 解密字符串
func (c *CryptoTool) DecryptString(encrypted string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	decrypted, err := c.DecryptAES(decoded)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// GenerateSignature 生成签名
func (c *CryptoTool) GenerateSignature(data string) string {
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名
func (c *CryptoTool) VerifySignature(data, signature string) bool {
	expectedSignature := c.GenerateSignature(data)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// GenerateRandomBytes 生成随机字节
func (c *CryptoTool) GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateRandomString 生成随机字符串
func (c *CryptoTool) GenerateRandomString(length int) (string, error) {
	bytes, err := c.GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// PasswordValidator 密码验证器
type PasswordValidator struct {
	minLength      int
	requireUpper   bool
	requireLower   bool
	requireNumber  bool
	requireSpecial bool
}

// NewPasswordValidator 创建密码验证器
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		minLength:      8,
		requireUpper:   true,
		requireLower:   true,
		requireNumber:  true,
		requireSpecial: true,
	}
}

// Validate 验证密码强度
func (v *PasswordValidator) Validate(password string) (bool, error) {
	if len(password) < v.minLength {
		return false, fmt.Errorf("password must be at least %d characters", v.minLength)
	}

	if v.requireUpper && !hasUpper(password) {
		return false, fmt.Errorf("password must contain uppercase letter")
	}

	if v.requireLower && !hasLower(password) {
		return false, fmt.Errorf("password must contain lowercase letter")
	}

	if v.requireNumber && !hasNumber(password) {
		return false, fmt.Errorf("password must contain number")
	}

	if v.requireSpecial && !hasSpecial(password) {
		return false, fmt.Errorf("password must contain special character")
	}

	return true, nil
}

func hasUpper(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasLower(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func hasNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func hasSpecial(s string) bool {
	special := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, r := range s {
		for _, sp := range special {
			if r == sp {
				return true
			}
		}
	}
	return false
}

// APIKeyGenerator API Key 生成器
type APIKeyGenerator struct {
	prefix string
}

// NewAPIKeyGenerator 创建 API Key 生成器
func NewAPIKeyGenerator(prefix string) *APIKeyGenerator {
	return &APIKeyGenerator{
		prefix: prefix,
	}
}

// Generate 生成 API Key
func (g *APIKeyGenerator) Generate() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	key := g.prefix + "_" + hex.EncodeToString(randomBytes)
	return key, nil
}

// HashAlgorithm 哈希算法接口
type HashAlgorithm interface {
	Hash(data []byte) []byte
	Verify(data []byte, hash []byte) bool
}

// SHA256Hash SHA256 哈希
type SHA256Hash struct{}

func (h *SHA256Hash) Hash(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	return hasher.Sum(nil)
}

func (h *SHA256Hash) Verify(data []byte, hash []byte) bool {
	expected := h.Hash(data)
	return hmac.Equal(expected, hash)
}

// HMACHash HMAC 哈希
type HMACHash struct {
	key []byte
}

func NewHMACHash(key string) *HMACHash {
	return &HMACHash{
		key: []byte(key),
	}
}

func (h *HMACHash) Hash(data []byte) []byte {
	hasher := hmac.New(sha256.New, h.key)
	hasher.Write(data)
	return hasher.Sum(nil)
}

func (h *HMACHash) Verify(data []byte, hash []byte) bool {
	expected := h.Hash(data)
	return hmac.Equal(expected, hash)
}

// EncryptionManager 加密管理器
type EncryptionManager struct {
	aesKey []byte
}

// NewEncryptionManager 创建加密管理器
func NewEncryptionManager(aesKey string) *EncryptionManager {
	// 确保密钥长度为 32 字节 (AES-256)
	key := []byte(aesKey)
	if len(key) < 32 {
		// 填充
		for len(key) < 32 {
			key = append(key, 0)
		}
	} else if len(key) > 32 {
		key = key[:32]
	}

	return &EncryptionManager{
		aesKey: key,
	}
}

// EncryptString 加密字符串
func (e *EncryptionManager) EncryptString(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.aesKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString 解密字符串
func (e *EncryptionManager) DecryptString(ciphertext string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.aesKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// MaskString 掩码字符串（用于日志）
func MaskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// IsSecurePassword 检查密码是否安全
func IsSecurePassword(password string) bool {
	validator := NewPasswordValidator()
	valid, _ := validator.Validate(password)
	return valid
}
