package ent

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// QueryExtension 查询扩展
type QueryExtension struct {
	db *sql.DB
}

// NewQueryExtension 创建查询扩展
func NewQueryExtension(db *sql.DB) *QueryExtension {
	return &QueryExtension{
		db: db,
	}
}

// BulkCreate 批量创建优化
func (e *QueryExtension) BulkCreate(ctx context.Context, items []interface{}, batchSize int) error {
	if len(items) == 0 {
		return nil
	}

	if batchSize == 0 {
		batchSize = 1000
	}

	// 分批处理
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]

		// 使用事务
		tx, err := e.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		// 执行批量插入
		// 简化实现：实际应该使用 prepared statement
		for _, item := range batch {
			_ = item // 使用 item
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return err
		}
	}

	return nil
}

// OptimizedQuery 查询优化
type OptimizedQuery struct {
	fields   []string
	joins    []string
	filters  []string
	orderBy  []string
	limit    *int
	offset   *int
	distinct bool
}

// NewOptimizedQuery 创建优化查询
func NewOptimizedQuery() *OptimizedQuery {
	return &OptimizedQuery{
		fields:  make([]string, 0),
		joins:   make([]string, 0),
		filters: make([]string, 0),
		orderBy: make([]string, 0),
	}
}

// Select 指定查询字段
func (q *OptimizedQuery) Select(fields ...string) *OptimizedQuery {
	q.fields = append(q.fields, fields...)
	return q
}

// Join 关联查询
func (q *OptimizedQuery) Join(table, on string) *OptimizedQuery {
	q.joins = append(q.joins, fmt.Sprintf("JOIN %s ON %s", table, on))
	return q
}

// Where 条件过滤
func (q *OptimizedQuery) Where(condition string) *OptimizedQuery {
	q.filters = append(q.filters, condition)
	return q
}

// OrderBy 排序
func (q *OptimizedQuery) OrderBy(field, direction string) *OptimizedQuery {
	q.orderBy = append(q.orderBy, fmt.Sprintf("%s %s", field, direction))
	return q
}

// Limit 限制数量
func (q *OptimizedQuery) Limit(n int) *OptimizedQuery {
	q.limit = &n
	return q
}

// Offset 偏移量
func (q *OptimizedQuery) Offset(n int) *OptimizedQuery {
	q.offset = &n
	return q
}

// Distinct 去重
func (q *OptimizedQuery) Distinct() *OptimizedQuery {
	q.distinct = true
	return q
}

// Build 构建 SQL
func (q *OptimizedQuery) Build() string {
	sql := "SELECT "

	if q.distinct {
		sql += "DISTINCT "
	}

	if len(q.fields) == 0 {
		sql += "*"
	} else {
		sql += q.fields[0]
		for i := 1; i < len(q.fields); i++ {
			sql += ", " + q.fields[i]
		}
	}

	if len(q.joins) > 0 {
		for _, join := range q.joins {
			sql += " " + join
		}
	}

	if len(q.filters) > 0 {
		sql += " WHERE " + q.filters[0]
		for i := 1; i < len(q.filters); i++ {
			sql += " AND " + q.filters[i]
		}
	}

	if len(q.orderBy) > 0 {
		sql += " ORDER BY " + q.orderBy[0]
		for i := 1; i < len(q.orderBy); i++ {
			sql += ", " + q.orderBy[i]
		}
	}

	if q.limit != nil {
		sql += fmt.Sprintf(" LIMIT %d", *q.limit)
	}

	if q.offset != nil {
		sql += fmt.Sprintf(" OFFSET %d", *q.offset)
	}

	return sql
}

// Execute 执行查询
func (q *OptimizedQuery) Execute(ctx context.Context, db *sql.DB) (*sql.Rows, error) {
	sql := q.Build()
	return db.QueryContext(ctx, sql)
}

// ConnectionPool 连接池配置
type ConnectionPool struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

// NewConnectionPool 创建连接池配置
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		maxOpenConns:    100,
		maxIdleConns:    10,
		connMaxLifetime: time.Hour,
		connMaxIdleTime: 10 * time.Minute,
	}
}

// WithMaxOpenConns 设置最大打开连接数
func (p *ConnectionPool) WithMaxOpenConns(n int) *ConnectionPool {
	p.maxOpenConns = n
	return p
}

// WithMaxIdleConns 设置最大空闲连接数
func (p *ConnectionPool) WithMaxIdleConns(n int) *ConnectionPool {
	p.maxIdleConns = n
	return p
}

// WithConnMaxLifetime 设置连接最大生命周期
func (p *ConnectionPool) WithConnMaxLifetime(d time.Duration) *ConnectionPool {
	p.connMaxLifetime = d
	return p
}

// WithConnMaxIdleTime 设置连接最大空闲时间
func (p *ConnectionPool) WithConnMaxIdleTime(d time.Duration) *ConnectionPool {
	p.connMaxIdleTime = d
	return p
}

// Apply 应用到数据库连接
func (p *ConnectionPool) Apply(db *sql.DB) {
	db.SetMaxOpenConns(p.maxOpenConns)
	db.SetMaxIdleConns(p.maxIdleConns)
	db.SetConnMaxLifetime(p.connMaxLifetime)
	db.SetConnMaxIdleTime(p.connMaxIdleTime)
}

// QueryMonitor 查询监控
type QueryMonitor struct {
	queries map[string]*QueryStats
	mu      sync.RWMutex
}

// QueryStats 查询统计
type QueryStats struct {
	Count   int64
	AvgTime time.Duration
	MaxTime time.Duration
	MinTime time.Duration
	Errors  int64
}

// NewQueryMonitor 创建查询监控
func NewQueryMonitor() *QueryMonitor {
	return &QueryMonitor{
		queries: make(map[string]*QueryStats),
	}
}

// Record 记录查询
func (m *QueryMonitor) Record(query string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, exists := m.queries[query]
	if !exists {
		stats = &QueryStats{
			MinTime: duration,
			MaxTime: duration,
		}
		m.queries[query] = stats
	}

	stats.Count++

	if stats.Count == 1 {
		stats.AvgTime = duration
	} else {
		stats.AvgTime = (stats.AvgTime*time.Duration(stats.Count-1) + duration) / time.Duration(stats.Count)
	}

	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}

	if err != nil {
		stats.Errors++
	}
}

// GetStats 获取统计
func (m *QueryMonitor) GetStats(query string) *QueryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queries[query]
}

// GetSlowQueries 获取慢查询
func (m *QueryMonitor) GetSlowQueries(threshold time.Duration) map[string]*QueryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slow := make(map[string]*QueryStats)
	for query, stats := range m.queries {
		if stats.AvgTime > threshold {
			slow[query] = stats
		}
	}
	return slow
}

// IndexAnalyzer 索引分析器
type IndexAnalyzer struct {
	tables map[string]TableInfo
}

// TableInfo 表信息
type TableInfo struct {
	Name    string
	Indexes []IndexInfo
	Columns []ColumnInfo
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// NewIndexAnalyzer 创建索引分析器
func NewIndexAnalyzer() *IndexAnalyzer {
	return &IndexAnalyzer{
		tables: make(map[string]TableInfo),
	}
}

// AddTable 添加表
func (a *IndexAnalyzer) AddTable(table TableInfo) {
	a.tables[table.Name] = table
}

// Analyze 分析索引使用情况
func (a *IndexAnalyzer) Analyze(query string) []string {
	// 简化实现：分析查询中的索引使用
	// 实际应该解析 SQL 并检查执行计划
	return []string{"建议在常用查询字段上添加索引"}
}

// GetMissingIndexes 获取缺失的索引
func (a *IndexAnalyzer) GetMissingIndexes() []string {
	var missing []string
	for _, table := range a.tables {
		// 检查是否有主键
		hasPrimary := false
		for _, idx := range table.Indexes {
			if idx.Type == "PRIMARY" {
				hasPrimary = true
				break
			}
		}
		if !hasPrimary {
			missing = append(missing, fmt.Sprintf("Table %s missing primary key", table.Name))
		}
	}
	return missing
}

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	monitors *QueryMonitor
	analyzer *IndexAnalyzer
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(monitor *QueryMonitor, analyzer *IndexAnalyzer) *QueryOptimizer {
	return &QueryOptimizer{
		monitors: monitor,
		analyzer: analyzer,
	}
}

// GetSuggestions 获取优化建议
func (o *QueryOptimizer) GetSuggestions() []string {
	suggestions := []string{}

	// 慢查询建议
	slowQueries := o.monitors.GetSlowQueries(1 * time.Second)
	if len(slowQueries) > 0 {
		suggestions = append(suggestions, "发现慢查询:")
		for query, stats := range slowQueries {
			suggestions = append(suggestions, fmt.Sprintf("  %s: 平均 %v", query, stats.AvgTime))
		}
	}

	// 索引建议
	missing := o.analyzer.GetMissingIndexes()
	if len(missing) > 0 {
		suggestions = append(suggestions, "缺失索引:")
		suggestions = append(suggestions, missing...)
	}

	return suggestions
}

// BatchProcessor 批量处理器
type BatchProcessor struct {
	batchSize int
	timeout   time.Duration
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor() *BatchProcessor {
	return &BatchProcessor{
		batchSize: 1000,
		timeout:   30 * time.Second,
	}
}

// WithBatchSize 设置批量大小
func (p *BatchProcessor) WithBatchSize(n int) *BatchProcessor {
	p.batchSize = n
	return p
}

// WithTimeout 设置超时
func (p *BatchProcessor) WithTimeout(d time.Duration) *BatchProcessor {
	p.timeout = d
	return p
}

// Process 批量处理
func (p *BatchProcessor) Process(ctx context.Context, items []interface{}, handler func([]interface{}) error) error {
	for i := 0; i < len(items); i += p.batchSize {
		end := i + p.batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]

		// 带超时的处理
		done := make(chan error, 1)
		go func() {
			done <- handler(batch)
		}()

		select {
		case err := <-done:
			if err != nil {
				return err
			}
		case <-time.After(p.timeout):
			return fmt.Errorf("batch processing timeout")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// QueryBuilder 查询构建器
type QueryBuilder struct {
	table   string
	aliases map[string]string
}

// NewQueryBuilder 创建查询构建器
func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:   table,
		aliases: make(map[string]string),
	}
}

// SelectWithAliases 带别名的字段选择
func (b *QueryBuilder) SelectWithAliases(fields map[string]string) string {
	result := "SELECT "
	first := true
	for field, alias := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s AS %s", field, alias)
		first = false
	}
	result += " FROM " + b.table
	return result
}

// BuildIn 构建 IN 查询
func (b *QueryBuilder) BuildIn(field string, values []interface{}) string {
	if len(values) == 0 {
		return "1=0"
	}

	result := field + " IN ("
	for i, value := range values {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("'%v'", value)
	}
	result += ")"
	return result
}

// BuildOr 构建 OR 查询
func (b *QueryBuilder) BuildOr(conditions []string) string {
	if len(conditions) == 0 {
		return "1=0"
	}

	result := "("
	for i, cond := range conditions {
		if i > 0 {
			result += " OR "
		}
		result += cond
	}
	result += ")"
	return result
}

// Pagination 分页查询
type Pagination struct {
	Page    int
	PerPage int
}

// NewPagination 创建分页
func NewPagination(page, perPage int) *Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 1000 {
		perPage = 1000
	}
	return &Pagination{
		Page:    page,
		PerPage: perPage,
	}
}

// Offset 计算偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit 获取限制
func (p *Pagination) Limit() int {
	return p.PerPage
}

// BuildLimit 构建 LIMIT 语句
func (p *Pagination) BuildLimit() string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", p.PerPage, p.Offset())
}

// TotalPages 计算总页数
func (p *Pagination) TotalPages(total int) int {
	if total == 0 {
		return 0
	}
	pages := total / p.PerPage
	if total%p.PerPage != 0 {
		pages++
	}
	return pages
}

// QueryResult 查询结果
type QueryResult struct {
	Data       interface{}
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

// QueryExecutor 查询执行器
type QueryExecutor struct {
	db      *sql.DB
	monitor *QueryMonitor
}

// NewQueryExecutor 创建查询执行器
func NewQueryExecutor(db *sql.DB, monitor *QueryMonitor) *QueryExecutor {
	return &QueryExecutor{
		db:      db,
		monitor: monitor,
	}
}

// ExecuteWithStats 带统计的查询执行
func (e *QueryExecutor) ExecuteWithStats(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := e.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if e.monitor != nil {
		e.monitor.Record(query, duration, err)
	}

	return rows, err
}

// ExecuteWithRetry 带重试的查询执行
func (e *QueryExecutor) ExecuteWithRetry(ctx context.Context, query string, maxRetries int, args ...interface{}) (*sql.Rows, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}

		rows, err := e.ExecuteWithStats(ctx, query, args...)
		if err == nil {
			return rows, nil
		}

		lastErr = err

		// 检查是否是可重试的错误
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryableError 检查是否是可重试的错误
func isRetryableError(err error) bool {
	// 简化实现：实际应该检查具体的错误类型
	// 如：连接错误、死锁等
	return false
}
