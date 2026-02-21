package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config SecurityConfig
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CORS            CORSConfig
	Helmet          bool
	IPWhitelist     []string
	IPBlacklist     []string
	RequestSize     int64
	EnableCSRF      bool
	EnableRateLimit bool
}

// CORSConfig CORS 配置
type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(config SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		config: config,
	}
}

// Chain 安全中间件链
func (s *SecurityMiddleware) Chain() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		h := next

		// IP 过滤
		h = s.ipFilterMiddleware(h)

		// 请求大小限制
		h = s.sizeLimitMiddleware(h)

		// CORS
		if s.config.CORS.Enabled {
			h = s.corsMiddleware(h)
		}

		// Helmet
		if s.config.Helmet {
			h = s.helmetMiddleware(h)
		}

		// CSRF (如果启用)
		if s.config.EnableCSRF {
			h = s.csrfMiddleware(h)
		}

		return h
	}
}

// ipFilterMiddleware IP 过滤中间件
func (s *SecurityMiddleware) ipFilterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		// 黑名单检查
		for _, black := range s.config.IPBlacklist {
			if ip == black {
				http.Error(w, "IP blocked", http.StatusForbidden)
				return
			}
		}

		// 白名单检查
		if len(s.config.IPWhitelist) > 0 {
			allowed := false
			for _, white := range s.config.IPWhitelist {
				if ip == white {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "IP not allowed", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// sizeLimitMiddleware 请求大小限制中间件
func (s *SecurityMiddleware) sizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.RequestSize > 0 && r.ContentLength > s.config.RequestSize {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware CORS 中间件
func (s *SecurityMiddleware) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 设置允许的来源
		if len(s.config.CORS.AllowedOrigins) > 0 {
			allowed := false
			for _, o := range s.config.CORS.AllowedOrigins {
				if o == "*" || o == origin {
					w.Header().Set("Access-Control-Allow-Origin", o)
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
		}

		// 设置其他 CORS 头
		if len(s.config.CORS.AllowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.CORS.AllowedMethods, ", "))
		}
		if len(s.config.CORS.AllowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.CORS.AllowedHeaders, ", "))
		}
		w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", s.config.CORS.AllowCredentials))
		w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", s.config.CORS.MaxAge))

		// 预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// helmetMiddleware Security Headers 中间件
func (s *SecurityMiddleware) helmetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 防止 MIME 类型嗅探
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// 防止点击劫持
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS 保护
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// HSTS (HTTPS)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer 策略
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 内容安全策略 (简化)
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware CSRF 防护中间件
func (s *SecurityMiddleware) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 只检查 POST, PUT, DELETE, PATCH
		method := r.Method
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			next.ServeHTTP(w, r)
			return
		}

		// 检查 Origin 和 Referer
		origin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")

		if origin == "" && referer == "" {
			http.Error(w, "CSRF validation failed: missing origin/referer", http.StatusForbidden)
			return
		}

		// 如果有 Origin，验证它
		if origin != "" {
			allowed := false
			for _, o := range s.config.CORS.AllowedOrigins {
				if o == "*" || strings.Contains(origin, o) {
					allowed = true
					break
				}
			}
			if !allowed {
				http.Error(w, "CSRF validation failed: invalid origin", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Crypto 加密工具
type Crypto struct {
	secretKey []byte
}

// NewCrypto 创建加密工具
func NewCrypto(secretKey string) *Crypto {
	return &Crypto{
		secretKey: []byte(secretKey),
	}
}

// HashPassword 哈希密码
func (c *Crypto) HashPassword(password string) (string, error) {
	// 使用 HMAC-SHA256
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyPassword 验证密码
func (c *Crypto) VerifyPassword(password, hash string) bool {
	expectedHash, err := c.HashPassword(password)
	if err != nil {
		return false
	}
	return expectedHash == hash
}

// Encrypt 数据加密（简化版）
func (c *Crypto) Encrypt(data []byte) ([]byte, error) {
	// 使用 XOR 简单加密（实际应使用 AES）
	encrypted := make([]byte, len(data))
	for i := range data {
		encrypted[i] = data[i] ^ c.secretKey[i%len(c.secretKey)]
	}
	return encrypted, nil
}

// Decrypt 数据解密
func (c *Crypto) Decrypt(encrypted []byte) ([]byte, error) {
	// XOR 解密
	decrypted := make([]byte, len(encrypted))
	for i := range encrypted {
		decrypted[i] = encrypted[i] ^ c.secretKey[i%len(c.secretKey)]
	}
	return decrypted, nil
}

// GenerateSignature 生成签名
func (c *Crypto) GenerateSignature(data string) string {
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名
func (c *Crypto) VerifySignature(data, signature string) bool {
	expectedSignature := c.GenerateSignature(data)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// JWTToken JWT Token 结构
type JWTToken struct {
	Header    map[string]interface{}
	Payload   map[string]interface{}
	Signature string
}

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey []byte
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// Generate 生成 JWT Token
func (j *JWTManager) Generate(payload map[string]interface{}) (string, error) {
	// 简化实现：不使用标准 JWT 格式
	// 实际应使用 github.com/golang-jwt/jwt/v5
	return "", fmt.Errorf("JWT not implemented, use HMAC signature instead")
}

// Verify 验证 JWT Token
func (j *JWTManager) Verify(token string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("JWT not implemented")
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	Burst             int
}

// RateLimiter 限流器
type RateLimiter struct {
	config  RateLimitConfig
	storage map[string]int
}

// NewRateLimiter 创建限流器
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:  config,
		storage: make(map[string]int),
	}
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow(key string) bool {
	// 简化实现：使用固定窗口
	count, exists := r.storage[key]
	if !exists {
		r.storage[key] = 1
		return true
	}

	if count >= r.config.RequestsPerMinute {
		return false
	}

	r.storage[key] = count + 1
	return true
}

// Reset 重置计数
func (r *RateLimiter) Reset(key string) {
	delete(r.storage, key)
}

// SecurityValidator 安全验证器
type SecurityValidator struct {
	crypto *Crypto
}

// NewSecurityValidator 创建安全验证器
func NewSecurityValidator(secretKey string) *SecurityValidator {
	return &SecurityValidator{
		crypto: NewCrypto(secretKey),
	}
}

// ValidateInput 验证输入
func (v *SecurityValidator) ValidateInput(input string) error {
	// 检查 SQL 注入
	if strings.Contains(strings.ToLower(input), "union") {
		return fmt.Errorf("potential SQL injection detected")
	}

	// 检查 XSS
	if strings.Contains(input, "<script") || strings.Contains(input, "javascript:") {
		return fmt.Errorf("potential XSS detected")
	}

	return nil
}

// SanitizeInput 清理输入
func (v *SecurityValidator) SanitizeInput(input string) string {
	// 移除危险字符
	replacer := strings.NewReplacer(
		"<", "<",
		">", ">",
		"\"", "\"",
		"'", "'",
	)
	return replacer.Replace(input)
}

// SecurityHeaders 安全头管理器
type SecurityHeaders struct {
	headers map[string]string
}

// NewSecurityHeaders 创建安全头管理器
func NewSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{
		headers: make(map[string]string),
	}
}

// Set 设置安全头
func (s *SecurityHeaders) Set(key, value string) {
	s.headers[key] = value
}

// Apply 应用到 ResponseWriter
func (s *SecurityHeaders) Apply(w http.ResponseWriter) {
	for key, value := range s.headers {
		w.Header().Set(key, value)
	}
}

// GetDefaultHeaders 获取默认安全头
func GetDefaultHeaders() map[string]string {
	return map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}
}

// SecurityMonitor 安全监控
type SecurityMonitor struct {
	events []SecurityEvent
}

// SecurityEvent 安全事件
type SecurityEvent struct {
	Timestamp int64
	Type      string
	IP        string
	Action    string
	Severity  string
}

// NewSecurityMonitor 创建安全监控
func NewSecurityMonitor() *SecurityMonitor {
	return &SecurityMonitor{
		events: make([]SecurityEvent, 0),
	}
}

// RecordEvent 记录安全事件
func (m *SecurityMonitor) RecordEvent(event SecurityEvent) {
	m.events = append(m.events, event)
}

// GetEvents 获取安全事件
func (m *SecurityMonitor) GetEvents() []SecurityEvent {
	return m.events
}

// GetEventsByType 按类型获取事件
func (m *SecurityMonitor) GetEventsByType(eventType string) []SecurityEvent {
	var result []SecurityEvent
	for _, event := range m.events {
		if event.Type == eventType {
			result = append(result, event)
		}
	}
	return result
}
