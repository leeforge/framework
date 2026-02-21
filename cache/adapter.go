package cache

import (
	"sync"
	"time"

	"github.com/leeforge/framework/auth/rbac"
)

// SimpleAdapter 简单的内存缓存适配器
// 实现 frame-core/auth/rbac.CacheAdapter 接口
type SimpleAdapter struct {
	cache map[string]*cacheItem
	mu    sync.RWMutex
}

type cacheItem struct {
	value     any
	expiresAt time.Time
}

// NewSimpleAdapter 创建简单缓存适配器
func NewSimpleAdapter() rbac.CacheAdapter {
	adapter := &SimpleAdapter{
		cache: make(map[string]*cacheItem),
	}

	// 启动后台清理过期缓存
	go adapter.cleanupExpired()

	return adapter
}

// Get 获取缓存
func (c *SimpleAdapter) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.cache[key]
	if !exists {
		return nil, &Error{Message: "cache not found"}
	}

	if time.Now().After(item.expiresAt) {
		return nil, &Error{Message: "cache expired"}
	}

	return item.value, nil
}

// Set 设置缓存
// ttl: 过期时间（秒）
func (c *SimpleAdapter) Set(key string, value interface{}, ttl int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}

	return nil
}

// Delete 删除缓存
func (c *SimpleAdapter) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
	return nil
}

// cleanupExpired 定期清理过期缓存
func (c *SimpleAdapter) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.cache {
			if now.After(item.expiresAt) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// Error 缓存错误
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
