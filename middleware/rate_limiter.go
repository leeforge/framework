package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter 限流器
type RateLimiter struct {
	backend BackendAdapter
	config  RateLimitConfig
}

// BackendAdapter 限流后端适配器
type BackendAdapter interface {
	CheckLimit(key string, limit int, window int64) (bool, error)
	Increment(key string, window int64) error
	GetUsage(key string) (int, error)
	Reset(key string) error
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	DefaultRate  int                 // 默认每分钟请求数
	DefaultDaily int                 // 默认每日请求数
	Strategies   map[string]Strategy // 按路径配置
	Burst        int                 // 突发流量支持
}

// Strategy 限流策略
type Strategy struct {
	Rate  int
	Daily int
	Burst int
}

// RedisBackend Redis 后端（简化版）
type RedisBackend struct {
	store map[string]int
	mu    sync.RWMutex
}

// NewRedisBackend 创建 Redis 后端
func NewRedisBackend() *RedisBackend {
	return &RedisBackend{
		store: make(map[string]int),
	}
}

// CheckLimit 检查限流
func (b *RedisBackend) CheckLimit(key string, limit int, window int64) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count, exists := b.store[key]
	if !exists {
		return true, nil
	}

	return count < limit, nil
}

// Increment 增加计数
func (b *RedisBackend) Increment(key string, window int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.store[key]++
	return nil
}

// GetUsage 获取使用量
func (b *RedisBackend) GetUsage(key string) (int, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count, exists := b.store[key]
	if !exists {
		return 0, nil
	}
	return count, nil
}

// Reset 重置计数
func (b *RedisBackend) Reset(key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.store, key)
	return nil
}

// NewRateLimiter 创建限流器
func NewRateLimiter(backend BackendAdapter, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		backend: backend,
		config:  config,
	}
}

// Middleware 限流中间件
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取 API Key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = "anonymous"
		}

		// 获取限流配置
		config := rl.getConfig(r.URL.Path)

		// 检查分钟限流
		minuteKey := fmt.Sprintf("rate:minute:%s", apiKey)
		if !rl.checkLimit(minuteKey, config.Rate, 60) {
			rl.writeError(w, 429, config.Rate, "minute")
			return
		}

		// 检查日限流
		dailyKey := fmt.Sprintf("rate:daily:%s", apiKey)
		if !rl.checkLimit(dailyKey, config.Daily, 86400) {
			rl.writeError(w, 429, config.Daily, "daily")
			return
		}

		// 检查突发流量
		if config.Burst > 0 {
			burstKey := fmt.Sprintf("rate:burst:%s", apiKey)
			if !rl.checkLimit(burstKey, config.Burst, 1) {
				rl.writeError(w, 429, config.Burst, "burst")
				return
			}
		}

		// 记录使用量
		rl.recordUsage(apiKey, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

// checkLimit 检查限流
func (rl *RateLimiter) checkLimit(key string, limit int, window int64) bool {
	allowed, err := rl.backend.CheckLimit(key, limit, window)
	if err != nil {
		return false
	}
	return allowed
}

// recordUsage 记录使用量
func (rl *RateLimiter) recordUsage(apiKey, path string) {
	// 分钟计数
	minuteKey := fmt.Sprintf("rate:minute:%s", apiKey)
	rl.backend.Increment(minuteKey, 60)

	// 日计数
	dailyKey := fmt.Sprintf("rate:daily:%s", apiKey)
	rl.backend.Increment(dailyKey, 86400)

	// 突发计数
	burstKey := fmt.Sprintf("rate:burst:%s", apiKey)
	rl.backend.Increment(burstKey, 1)
}

// getConfig 获取路径对应的限流配置
func (rl *RateLimiter) getConfig(path string) Strategy {
	// 查找最匹配的路径
	for pattern, strategy := range rl.config.Strategies {
		if matchPath(pattern, path) {
			return strategy
		}
	}

	// 返回默认配置
	return Strategy{
		Rate:  rl.config.DefaultRate,
		Daily: rl.config.DefaultDaily,
		Burst: rl.config.Burst,
	}
}

// matchPath 路径匹配
func matchPath(pattern, path string) bool {
	// 简化实现：精确匹配或前缀匹配
	if pattern == path {
		return true
	}
	return len(path) >= len(pattern) && path[:len(pattern)] == pattern
}

// writeError 写入限流错误
func (rl *RateLimiter) writeError(w http.ResponseWriter, status int, limit int, window string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":429,"message":"Rate limit exceeded for %s window","limit":%d}}`, window, limit)
}

// ResetKey 重置指定 Key 的限流
func (rl *RateLimiter) ResetKey(apiKey string) error {
	minuteKey := fmt.Sprintf("rate:minute:%s", apiKey)
	dailyKey := fmt.Sprintf("rate:daily:%s", apiKey)
	burstKey := fmt.Sprintf("rate:burst:%s", apiKey)

	if err := rl.backend.Reset(minuteKey); err != nil {
		return err
	}
	if err := rl.backend.Reset(dailyKey); err != nil {
		return err
	}
	if err := rl.backend.Reset(burstKey); err != nil {
		return err
	}

	return nil
}

// GetUsage 获取 API Key 的使用情况
func (rl *RateLimiter) GetUsage(apiKey string) (minute, daily, burst int, err error) {
	minuteKey := fmt.Sprintf("rate:minute:%s", apiKey)
	dailyKey := fmt.Sprintf("rate:daily:%s", apiKey)
	burstKey := fmt.Sprintf("rate:burst:%s", apiKey)

	minute, err = rl.backend.GetUsage(minuteKey)
	if err != nil {
		return 0, 0, 0, err
	}

	daily, err = rl.backend.GetUsage(dailyKey)
	if err != nil {
		return 0, 0, 0, err
	}

	burst, err = rl.backend.GetUsage(burstKey)
	if err != nil {
		return 0, 0, 0, err
	}

	return minute, daily, burst, nil
}

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	backend BackendAdapter
	window  time.Duration
	limit   int
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(backend BackendAdapter, window time.Duration, limit int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		backend: backend,
		window:  window,
		limit:   limit,
	}
}

// Allow 检查是否允许请求
func (l *SlidingWindowLimiter) Allow(key string) (bool, error) {
	// 滑动窗口逻辑
	// 简化实现：使用固定窗口
	return l.backend.CheckLimit(key, l.limit, int64(l.window.Seconds()))
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	backend    BackendAdapter
	capacity   int
	refillRate float64 // 每秒 refill 数量
	mu         sync.Mutex
}

// NewTokenBucketLimiter 创建令牌桶限流器
func NewTokenBucketLimiter(backend BackendAdapter, capacity int, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		backend:    backend,
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// Allow 检查是否允许请求
func (l *TokenBucketLimiter) Allow(key string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取当前令牌数
	current, err := l.backend.GetUsage(key)
	if err != nil {
		return false, err
	}

	// 如果没有令牌，拒绝
	if current <= 0 {
		return false, nil
	}

	// 消耗一个令牌
	if err := l.backend.Increment(key, 0); err != nil {
		return false, err
	}

	return true, nil
}

// Refill 补充令牌（需要定时调用）
func (l *TokenBucketLimiter) Refill(key string, tokens int) error {
	current, err := l.backend.GetUsage(key)
	if err != nil {
		return err
	}

	newCount := current + tokens
	if newCount > l.capacity {
		newCount = l.capacity
	}

	// 重置为新值
	if err := l.backend.Reset(key); err != nil {
		return err
	}

	for i := 0; i < newCount; i++ {
		if err := l.backend.Increment(key, 0); err != nil {
			return err
		}
	}

	return nil
}

// RateLimitMiddleware 限流中间件工厂
type RateLimitMiddleware struct {
	limiter *RateLimiter
}

// NewRateLimitMiddleware 创建限流中间件工厂
func NewRateLimitMiddleware(backend BackendAdapter, config RateLimitConfig) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: NewRateLimiter(backend, config),
	}
}

// Middleware 获取中间件
func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return m.limiter.Middleware(next)
}

// WithPath 针对特定路径的限流
func (m *RateLimitMiddleware) WithPath(path string, strategy Strategy) *RateLimitMiddleware {
	if m.limiter.config.Strategies == nil {
		m.limiter.config.Strategies = make(map[string]Strategy)
	}
	m.limiter.config.Strategies[path] = strategy
	return m
}

// RateLimitHandler 限流管理处理器
type RateLimitHandler struct {
	limiter *RateLimiter
}

// NewRateLimitHandler 创建限流管理处理器
func NewRateLimitHandler(limiter *RateLimiter) *RateLimitHandler {
	return &RateLimitHandler{
		limiter: limiter,
	}
}

// GetUsage 获取使用情况
func (h *RateLimitHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		apiKey = r.Header.Get("X-API-Key")
	}

	if apiKey == "" {
		http.Error(w, "API key required", http.StatusBadRequest)
		return
	}

	minute, daily, burst, err := h.limiter.GetUsage(apiKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"api_key":"%s","minute":%d,"daily":%d,"burst":%d}`, apiKey, minute, daily, burst)
}

// ResetUsage 重置使用情况
func (h *RateLimitHandler) ResetUsage(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("api_key")
	if apiKey == "" {
		apiKey = r.Header.Get("X-API-Key")
	}

	if apiKey == "" {
		http.Error(w, "API key required", http.StatusBadRequest)
		return
	}

	if err := h.limiter.ResetKey(apiKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message":"reset successful","api_key":"%s"}`, apiKey)
}

// RegisterRateLimitRoutes 注册限流管理路由
func RegisterRateLimitRoutes(mux *http.ServeMux, handler *RateLimitHandler) {
	mux.HandleFunc("/admin/rate-limit/usage", handler.GetUsage)
	mux.HandleFunc("/admin/rate-limit/reset", handler.ResetUsage)
}
