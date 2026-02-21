package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheStrategy 缓存策略
type CacheStrategy struct {
	config CacheConfig
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL         map[string]time.Duration
	MaxSize     int
	Compression bool
	Prefix      string
}

// NewCacheStrategy 创建缓存策略
func NewCacheStrategy(config CacheConfig) *CacheStrategy {
	return &CacheStrategy{
		config: config,
	}
}

// Key 生成缓存键
func (s *CacheStrategy) Key(prefix string, keys ...string) string {
	fullKey := prefix
	for _, key := range keys {
		fullKey += ":" + key
	}
	return s.config.Prefix + fullKey
}

// GetTTL 获取指定类型的 TTL
func (s *CacheStrategy) GetTTL(cacheType string) time.Duration {
	if ttl, exists := s.config.TTL[cacheType]; exists {
		return ttl
	}
	return 10 * time.Minute // 默认 10 分钟
}

// CacheAdapter 缓存适配器接口
type CacheAdapter interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) bool
}

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	L1 *sync.Map    // 本地内存
	L2 CacheAdapter // 二级缓存 (Redis 或其他)
	L3 LoaderFunc   // 三级缓存 (数据库加载器)
}

// LoaderFunc 数据加载函数
type LoaderFunc func(ctx context.Context) (interface{}, error)

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(l2 CacheAdapter, l3 LoaderFunc) *MultiLevelCache {
	return &MultiLevelCache{
		L1: &sync.Map{},
		L2: l2,
		L3: l3,
	}
}

// Get 获取缓存，支持自动加载
func (m *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, error) {
	// 1. L1 缓存 (内存)
	if val, ok := m.L1.Load(key); ok {
		return val, nil
	}

	// 2. L2 缓存
	if m.L2 != nil {
		val, err := m.L2.Get(key)
		if err == nil {
			// 回写 L1
			m.L1.Store(key, val)
			return val, nil
		}
	}

	// 3. L3 自动加载
	if m.L3 != nil {
		result, err := m.L3(ctx)
		if err != nil {
			return nil, err
		}

		// 回写缓存
		m.Set(ctx, key, result)
		return result, nil
	}

	return nil, fmt.Errorf("cache miss")
}

// Set 设置缓存 (L1 + L2)
func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}) error {
	// L1
	m.L1.Store(key, value)

	// L2
	if m.L2 != nil {
		return m.L2.Set(key, value, 10*time.Minute)
	}

	return nil
}

// Delete 删除缓存
func (m *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// L1
	m.L1.Delete(key)

	// L2
	if m.L2 != nil {
		return m.L2.Delete(key)
	}

	return nil
}

// Clear 清空缓存
func (m *MultiLevelCache) Clear(ctx context.Context) error {
	// 清空 L1
	m.L1 = &sync.Map{}

	// 清空 L2
	if m.L2 != nil {
		// 简化实现：需要 L2 支持 Clear
		// 实际应该遍历删除
	}

	return nil
}

// CacheAside 缓存旁路模式
type CacheAside struct {
	cache *MultiLevelCache
}

// NewCacheAside 创建缓存旁路
func NewCacheAside(cache *MultiLevelCache) *CacheAside {
	return &CacheAside{
		cache: cache,
	}
}

// Read 读取数据 (Cache-Aside)
func (c *CacheAside) Read(ctx context.Context, key string) (interface{}, error) {
	return c.cache.Get(ctx, key)
}

// Write 写入数据 (Cache-Aside)
func (c *CacheAside) Write(ctx context.Context, key string, value interface{}) error {
	return c.cache.Set(ctx, key, value)
}

// WriteThrough 写穿透
type WriteThrough struct {
	cache *MultiLevelCache
	store StoreAdapter
}

// StoreAdapter 存储适配器
type StoreAdapter interface {
	Save(ctx context.Context, key string, value interface{}) error
}

// NewWriteThrough 创建写穿透
func NewWriteThrough(cache *MultiLevelCache, store StoreAdapter) *WriteThrough {
	return &WriteThrough{
		cache: cache,
		store: store,
	}
}

// Write 写入数据 (Write-Through)
func (w *WriteThrough) Write(ctx context.Context, key string, value interface{}) error {
	// 先写缓存
	if err := w.cache.Set(ctx, key, value); err != nil {
		return err
	}

	// 再写存储
	return w.store.Save(ctx, key, value)
}

// WriteBack 写回
type WriteBack struct {
	cache         *MultiLevelCache
	store         StoreAdapter
	writeQueue    chan writeJob
	flushInterval time.Duration
}

type writeJob struct {
	key   string
	value interface{}
}

// NewWriteBack 创建写回
func NewWriteBack(cache *MultiLevelCache, store StoreAdapter, flushInterval time.Duration) *WriteBack {
	wb := &WriteBack{
		cache:         cache,
		store:         store,
		writeQueue:    make(chan writeJob, 100),
		flushInterval: flushInterval,
	}
	go wb.flushWorker()
	return wb
}

// Write 写入数据 (Write-Back)
func (w *WriteBack) Write(ctx context.Context, key string, value interface{}) error {
	// 只写缓存
	if err := w.cache.Set(ctx, key, value); err != nil {
		return err
	}

	// 加入写队列
	select {
	case w.writeQueue <- writeJob{key: key, value: value}:
		return nil
	default:
		return fmt.Errorf("write queue full")
	}
}

// flushWorker 后台刷写线程
func (w *WriteBack) flushWorker() {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for range ticker.C {
		// 批量刷写
		// 简化实现：这里应该从队列中取出并写入存储
	}
}

// CacheStrategyBuilder 缓存策略构建器
type CacheStrategyBuilder struct {
	config CacheConfig
}

// NewCacheStrategyBuilder 创建构建器
func NewCacheStrategyBuilder() *CacheStrategyBuilder {
	return &CacheStrategyBuilder{
		config: CacheConfig{
			TTL:         make(map[string]time.Duration),
			MaxSize:     1000,
			Compression: false,
			Prefix:      "app:",
		},
	}
}

// WithTTL 设置 TTL
func (b *CacheStrategyBuilder) WithTTL(cacheType string, ttl time.Duration) *CacheStrategyBuilder {
	b.config.TTL[cacheType] = ttl
	return b
}

// WithMaxSize 设置最大大小
func (b *CacheStrategyBuilder) WithMaxSize(size int) *CacheStrategyBuilder {
	b.config.MaxSize = size
	return b
}

// WithCompression 启用压缩
func (b *CacheStrategyBuilder) WithCompression(enabled bool) *CacheStrategyBuilder {
	b.config.Compression = enabled
	return b
}

// WithPrefix 设置前缀
func (b *CacheStrategyBuilder) WithPrefix(prefix string) *CacheStrategyBuilder {
	b.config.Prefix = prefix
	return b
}

// Build 构建策略
func (b *CacheStrategyBuilder) Build() *CacheStrategy {
	return NewCacheStrategy(b.config)
}

// CacheMetrics 缓存指标
type CacheMetrics struct {
	Hits   int64
	Misses int64
	Evicts int64
	Sets   int64
	Gets   int64
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics map[string]*CacheMetrics
	mu      sync.RWMutex
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*CacheMetrics),
	}
}

// RecordHit 记录命中
func (c *MetricsCollector) RecordHit(cacheType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.metrics[cacheType]; !exists {
		c.metrics[cacheType] = &CacheMetrics{}
	}
	c.metrics[cacheType].Hits++
	c.metrics[cacheType].Gets++
}

// RecordMiss 记录未命中
func (c *MetricsCollector) RecordMiss(cacheType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.metrics[cacheType]; !exists {
		c.metrics[cacheType] = &CacheMetrics{}
	}
	c.metrics[cacheType].Misses++
	c.metrics[cacheType].Gets++
}

// RecordSet 记录设置
func (c *MetricsCollector) RecordSet(cacheType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.metrics[cacheType]; !exists {
		c.metrics[cacheType] = &CacheMetrics{}
	}
	c.metrics[cacheType].Sets++
}

// RecordEvict 记录驱逐
func (c *MetricsCollector) RecordEvict(cacheType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.metrics[cacheType]; !exists {
		c.metrics[cacheType] = &CacheMetrics{}
	}
	c.metrics[cacheType].Evicts++
}

// GetMetrics 获取指标
func (c *MetricsCollector) GetMetrics(cacheType string) *CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if metrics, exists := c.metrics[cacheType]; exists {
		return metrics
	}
	return &CacheMetrics{}
}

// GetHitRate 获取命中率
func (c *MetricsCollector) GetHitRate(cacheType string) float64 {
	metrics := c.GetMetrics(cacheType)
	if metrics.Gets == 0 {
		return 0
	}
	return float64(metrics.Hits) / float64(metrics.Gets) * 100
}

// CacheProtection 缓存保护机制
type CacheProtection struct {
	cache *MultiLevelCache
}

// NewCacheProtection 创建缓存保护
func NewCacheProtection(cache *MultiLevelCache) *CacheProtection {
	return &CacheProtection{
		cache: cache,
	}
}

// PreventCachePenetration 防止缓存穿透
func (p *CacheProtection) PreventCachePenetration(ctx context.Context, key string) (interface{}, error) {
	// 简化实现：使用空值缓存
	// 实际应该检查空值标记
	return p.cache.Get(ctx, key)
}

// PreventCacheAvalanche 防止缓存雪崩
func (p *CacheProtection) PreventCacheAvalanche(keys []string, ttl time.Duration) error {
	// 为每个 key 添加随机 TTL
	// 简化实现：需要 L2 支持随机 TTL
	return nil
}

// PreventCacheBreakdown 防止缓存击穿
func (p *CacheProtection) PreventCacheBreakdown(ctx context.Context, key string) (interface{}, error) {
	// 简化实现：使用互斥锁
	// 需要分布式锁支持
	return p.cache.Get(ctx, key)
}

// CacheWarmup 缓存预热
type CacheWarmup struct {
	cache  *MultiLevelCache
	loader LoaderFunc
}

// NewCacheWarmup 创建缓存预热
func NewCacheWarmup(cache *MultiLevelCache, loader LoaderFunc) *CacheWarmup {
	return &CacheWarmup{
		cache:  cache,
		loader: loader,
	}
}

// Warmup 预热缓存
func (w *CacheWarmup) Warmup(ctx context.Context, keys []string) error {
	for _, key := range keys {
		_, err := w.cache.Get(ctx, key)
		if err != nil {
			return err
		}
	}
	return nil
}

// CacheEvictionPolicy 缓存淘汰策略接口
type CacheEvictionPolicy interface {
	Evict(cache *MultiLevelCache) error
}

// LFUPolicy LFU 淘汰策略
type LFUPolicy struct {
	frequency map[string]int
	mu        sync.RWMutex
}

// NewLFUPolicy 创建 LFU 策略
func NewLFUPolicy() *LFUPolicy {
	return &LFUPolicy{
		frequency: make(map[string]int),
	}
}

// RecordAccess 记录访问
func (p *LFUPolicy) RecordAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.frequency[key]++
}

// Evict 淘汰
func (p *LFUPolicy) Evict(cache *MultiLevelCache) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 找到频率最低的 key
	var minKey string
	minFreq := -1

	for key, freq := range p.frequency {
		if minFreq == -1 || freq < minFreq {
			minFreq = freq
			minKey = key
		}
	}

	if minKey != "" {
		return cache.Delete(context.Background(), minKey)
	}

	return nil
}

// LRUPolicy LRU 淘汰策略
type LRUPolicy struct {
	accessOrder []string
	mu          sync.RWMutex
}

// NewLRUPolicy 创建 LRU 策略
func NewLRUPolicy() *LRUPolicy {
	return &LRUPolicy{
		accessOrder: make([]string, 0),
	}
}

// RecordAccess 记录访问
func (p *LRUPolicy) RecordAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 移除已存在的
	for i, k := range p.accessOrder {
		if k == key {
			p.accessOrder = append(p.accessOrder[:i], p.accessOrder[i+1:]...)
			break
		}
	}

	// 添加到末尾
	p.accessOrder = append(p.accessOrder, key)
}

// Evict 淘汰
func (p *LRUPolicy) Evict(cache *MultiLevelCache) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.accessOrder) > 0 {
		oldestKey := p.accessOrder[0]
		return cache.Delete(context.Background(), oldestKey)
	}

	return nil
}

// TTLCache TTL 缓存
type TTLCache struct {
	cache *sync.Map
	ttl   time.Duration
	mu    sync.RWMutex
}

// TTLItem 带 TTL 的缓存项
type TTLItem struct {
	Value      interface{}
	Expiration int64
}

// NewTTLCache 创建 TTL 缓存
func NewTTLCache(ttl time.Duration) *TTLCache {
	return &TTLCache{
		cache: &sync.Map{},
		ttl:   ttl,
	}
}

// Get 获取带 TTL 检查
func (c *TTLCache) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.cache.Load(key)
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	ttlItem := item.(TTLItem)
	if time.Now().UnixNano() > ttlItem.Expiration {
		c.cache.Delete(key)
		return nil, fmt.Errorf("key expired")
	}

	return ttlItem.Value, nil
}

// Set 设置带 TTL
func (c *TTLCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := time.Now().Add(c.ttl).UnixNano()
	c.cache.Store(key, TTLItem{
		Value:      value,
		Expiration: expiration,
	})
}

// Cleanup 清理过期项
func (c *TTLCache) Cleanup() {
	c.cache.Range(func(key, value interface{}) bool {
		ttlItem := value.(TTLItem)
		if time.Now().UnixNano() > ttlItem.Expiration {
			c.cache.Delete(key)
		}
		return true
	})
}

// CacheStats 缓存统计
type CacheStats struct {
	TotalHits    int64
	TotalMisses  int64
	TotalSets    int64
	TotalDeletes int64
	CurrentSize  int
	HitRate      float64
}

// CacheMonitor 缓存监控
type CacheMonitor struct {
	stats CacheStats
	mu    sync.RWMutex
}

// NewCacheMonitor 创建监控
func NewCacheMonitor() *CacheMonitor {
	return &CacheMonitor{}
}

// RecordHit 记录命中
func (m *CacheMonitor) RecordHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalHits++
}

// RecordMiss 记录未命中
func (m *CacheMonitor) RecordMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalMisses++
}

// RecordSet 记录设置
func (m *CacheMonitor) RecordSet() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalSets++
}

// RecordDelete 记录删除
func (m *CacheMonitor) RecordDelete() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalDeletes++
}

// GetStats 获取统计
func (m *CacheMonitor) GetStats() CacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.stats.TotalHits + m.stats.TotalMisses
	if total > 0 {
		m.stats.HitRate = float64(m.stats.TotalHits) / float64(total) * 100
	}

	return m.stats
}

// Reset 重置统计
func (m *CacheMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = CacheStats{}
}
