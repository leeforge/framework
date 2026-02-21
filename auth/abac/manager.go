package abac

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ABACManager ABAC 管理器
type ABACManager struct {
	expressions map[string]*Expression
	mu          sync.RWMutex
	cache       CacheAdapter
}

// CacheAdapter 缓存适配器
type CacheAdapter interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, ttl int64) error
	Delete(key string) error
}

// Expression 表达式
type Expression struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Condition   string    `json:"condition"`
	Effect      string    `json:"effect"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

// Attributes 属性结构
type Attributes struct {
	User     map[string]interface{} `json:"user"`
	Resource map[string]interface{} `json:"resource"`
	Context  map[string]interface{} `json:"context"`
}

// PolicyRule 策略规则
type PolicyRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Condition   string    `json:"condition"`
	Effect      string    `json:"effect"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewABACManager 创建 ABAC 管理器
func NewABACManager(cache CacheAdapter) *ABACManager {
	return &ABACManager{
		expressions: make(map[string]*Expression),
		cache:       cache,
	}
}

// CreatePolicy 创建策略
func (m *ABACManager) CreatePolicy(ctx context.Context, rule PolicyRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.expressions[rule.ID]; exists {
		return fmt.Errorf("policy %s already exists", rule.ID)
	}

	expr := &Expression{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Condition:   rule.Condition,
		Effect:      rule.Effect,
		Priority:    rule.Priority,
		CreatedAt:   time.Now(),
	}

	m.expressions[rule.ID] = expr
	return nil
}

// UpdatePolicy 更新策略
func (m *ABACManager) UpdatePolicy(ctx context.Context, id string, rule PolicyRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.expressions[id]; !exists {
		return fmt.Errorf("policy %s not found", id)
	}

	expr := &Expression{
		ID:          id,
		Name:        rule.Name,
		Description: rule.Description,
		Condition:   rule.Condition,
		Effect:      rule.Effect,
		Priority:    rule.Priority,
		CreatedAt:   m.expressions[id].CreatedAt,
	}

	m.expressions[id] = expr
	return nil
}

// DeletePolicy 删除策略
func (m *ABACManager) DeletePolicy(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.expressions[id]; !exists {
		return fmt.Errorf("policy %s not found", id)
	}

	delete(m.expressions, id)
	return nil
}

// GetPolicies 获取所有策略
func (m *ABACManager) GetPolicies(ctx context.Context) ([]*PolicyRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*PolicyRule, 0, len(m.expressions))
	for _, expr := range m.expressions {
		rules = append(rules, &PolicyRule{
			ID:          expr.ID,
			Name:        expr.Name,
			Description: expr.Description,
			Condition:   expr.Condition,
			Effect:      expr.Effect,
			Priority:    expr.Priority,
			CreatedAt:   expr.CreatedAt,
		})
	}

	return rules, nil
}

// CheckPermission 权限检查
func (m *ABACManager) CheckPermission(
	ctx context.Context,
	userAttrs map[string]interface{},
	resourceAttrs map[string]interface{},
	action string,
	contextAttrs map[string]interface{},
) (bool, error) {
	// 构建属性对象
	attrs := Attributes{
		User:     userAttrs,
		Resource: resourceAttrs,
		Context:  contextAttrs,
	}

	// 添加动作到上下文
	attrs.Context["action"] = action

	// 按优先级排序策略
	sortedExprs := m.sortExpressionsByPriority()

	// 评估每个策略
	for _, expr := range sortedExprs {
		result, err := m.EvaluateCondition(expr.Condition, attrs)
		if err != nil {
			// 记录错误但继续
			continue
		}

		if result {
			// 匹配策略，返回效果
			return expr.Effect == "allow", nil
		}
	}

	// 默认拒绝
	return false, nil
}

// EvaluateCondition 评估条件表达式（简化版）
func (m *ABACManager) EvaluateCondition(expr string, attrs Attributes) (bool, error) {
	// 简化的条件评估
	// 实际使用时应该引入表达式引擎如 govaluate

	// 支持的基本操作:
	// 1. user.role == "admin"
	// 2. resource.owner == user.id
	// 3. context.action == "read"

	// 解析简单条件
	// 格式: field.path operator value

	parts := strings.Split(expr, " ")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid condition format: %s", expr)
	}

	left := parts[0]
	operator := parts[1]
	right := parts[2]

	// 获取左边的值
	leftValue, err := m.getAttributeValue(left, attrs)
	if err != nil {
		return false, err
	}

	// 去除右边的引号
	right = strings.Trim(right, "\"'")

	// 比较
	switch operator {
	case "==":
		return fmt.Sprintf("%v", leftValue) == right, nil
	case "!=":
		return fmt.Sprintf("%v", leftValue) != right, nil
	case "in":
		// 简化的 in 操作
		return strings.Contains(fmt.Sprintf("%v", leftValue), right), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// getAttributeValue 获取属性值
func (m *ABACManager) getAttributeValue(path string, attrs Attributes) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	scope := parts[0]
	key := parts[1]

	switch scope {
	case "user":
		return attrs.User[key], nil
	case "resource":
		return attrs.Resource[key], nil
	case "context":
		return attrs.Context[key], nil
	default:
		return nil, fmt.Errorf("unknown scope: %s", scope)
	}
}

// sortExpressionsByPriority 按优先级排序表达式
func (m *ABACManager) sortExpressionsByPriority() []*Expression {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exprs := make([]*Expression, 0, len(m.expressions))
	for _, expr := range m.expressions {
		exprs = append(exprs, expr)
	}

	// 按优先级降序排序
	for i := 0; i < len(exprs)-1; i++ {
		for j := i + 1; j < len(exprs); j++ {
			if exprs[i].Priority < exprs[j].Priority {
				exprs[i], exprs[j] = exprs[j], exprs[i]
			}
		}
	}

	return exprs
}

// AddExpression 添加表达式
func (m *ABACManager) AddExpression(id, name, condition, effect string, priority int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.expressions[id] = &Expression{
		ID:        id,
		Name:      name,
		Condition: condition,
		Effect:    effect,
		Priority:  priority,
		CreatedAt: time.Now(),
	}

	return nil
}

// RemoveExpression 移除表达式
func (m *ABACManager) RemoveExpression(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.expressions, id)
	return nil
}

// GetExpression 获取表达式
func (m *ABACManager) GetExpression(id string) (*Expression, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	expr, exists := m.expressions[id]
	if !exists {
		return nil, fmt.Errorf("expression %s not found", id)
	}

	return expr, nil
}

// ValidatePolicy 验证策略语法
func (m *ABACManager) ValidatePolicy(condition string) error {
	parts := strings.Split(condition, " ")
	if len(parts) != 3 {
		return fmt.Errorf("invalid condition format")
	}
	return nil
}

// TestPolicy 测试策略
func (m *ABACManager) TestPolicy(condition string, attrs Attributes) (bool, error) {
	return m.EvaluateCondition(condition, attrs)
}

// BatchCheckPermission 批量权限检查
func (m *ABACManager) BatchCheckPermission(
	ctx context.Context,
	userAttrs map[string]interface{},
	resourceAttrsList []map[string]interface{},
	action string,
	contextAttrs map[string]interface{},
) ([]bool, error) {
	results := make([]bool, len(resourceAttrsList))

	for i, resourceAttrs := range resourceAttrsList {
		result, err := m.CheckPermission(ctx, userAttrs, resourceAttrs, action, contextAttrs)
		if err != nil {
			return nil, fmt.Errorf("check failed for resource %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetMatchingPolicies 获取匹配的策略
func (m *ABACManager) GetMatchingPolicies(
	userAttrs map[string]interface{},
	resourceAttrs map[string]interface{},
	action string,
	contextAttrs map[string]interface{},
) ([]*Expression, error) {
	attrs := Attributes{
		User:     userAttrs,
		Resource: resourceAttrs,
		Context:  contextAttrs,
	}
	attrs.Context["action"] = action

	var matching []*Expression
	for _, expr := range m.expressions {
		result, err := m.EvaluateCondition(expr.Condition, attrs)
		if err != nil {
			continue
		}
		if result {
			matching = append(matching, expr)
		}
	}

	return matching, nil
}

// ClearPolicies 清空所有策略
func (m *ABACManager) ClearPolicies() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.expressions = make(map[string]*Expression)
	return nil
}

// GetPolicyCount 获取策略数量
func (m *ABACManager) GetPolicyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.expressions)
}
