package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"

	casbinlib "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	casbinadapter "github.com/leeforge/framework/auth/casbin"
)

// RBACManager RBAC 管理器
type RBACManager struct {
	enforcer *casbinlib.Enforcer
	adapter  *casbinadapter.EntAdapter
	cache    CacheAdapter
	mu       sync.RWMutex
}

// CacheAdapter 缓存适配器接口
type CacheAdapter interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, ttl int64) error
	Delete(key string) error
}

// Role 角色
type Role struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission 权限
type Permission struct {
	RoleCode  string    `json:"role_code"`
	Domain    string    `json:"domain"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

// Policy 策略
type Policy struct {
	Subject string `json:"subject"`
	Domain  string `json:"domain"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

// NewRBACManager 创建 RBAC 管理器
func NewRBACManager(adapter *casbinadapter.EntAdapter, cache CacheAdapter) (*RBACManager, error) {
	// 创建 Casbin 模型
	modelText := `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act
p2 = sub, dom, resource_key, scope_type, scope_value

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
`

	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// 创建 Enforcer
	enforcer, err := casbinlib.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	return &RBACManager{
		enforcer: enforcer,
		adapter:  adapter,
		cache:    cache,
	}, nil
}

// Enforcer returns the underlying Casbin enforcer for advanced domain-aware use cases.
func (m *RBACManager) Enforcer() *casbinlib.Enforcer {
	return m.enforcer
}

// CreateRole 创建角色
func (m *RBACManager) CreateRole(ctx context.Context, code, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查角色是否已存在
	if m.enforcer.HasGroupingPolicy(code) {
		return fmt.Errorf("role %s already exists", code)
	}

	// 添加角色到策略
	// 注意：Casbin 中角色通过 g 策略管理，这里我们只记录元数据
	return nil
}

// DeleteRole 删除角色
func (m *RBACManager) DeleteRole(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除角色的所有权限
	_, err := m.enforcer.RemoveFilteredPolicy(0, code)
	if err != nil {
		return err
	}

	// 删除角色继承关系
	_, err = m.enforcer.RemoveFilteredGroupingPolicy(0, code)
	return err
}

// GetRoles 获取所有角色
func (m *RBACManager) GetRoles(ctx context.Context) ([]*Role, error) {
	// 从缓存获取
	if m.cache != nil {
		if cached, err := m.cache.Get("rbac:roles"); err == nil {
			if roles, ok := cached.([]*Role); ok {
				return roles, nil
			}
		}
	}

	// 从 Casbin 获取所有角色
	groupingPolicy := m.enforcer.GetGroupingPolicy()
	roleMap := make(map[string]bool)
	for _, policy := range groupingPolicy {
		if len(policy) >= 2 {
			roleMap[policy[1]] = true
		}
	}

	roles := make([]*Role, 0, len(roleMap))
	for code := range roleMap {
		roles = append(roles, &Role{
			Code: code,
			Name: code,
		})
	}

	// 缓存结果
	if m.cache != nil {
		m.cache.Set("rbac:roles", roles, 300)
	}

	return roles, nil
}

// AddPermission 添加域内权限
func (m *RBACManager) AddPermission(ctx context.Context, roleCode, domain, resource, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 添加权限策略
	_, err := m.enforcer.AddPolicy(roleCode, domain, resource, action)
	if err != nil {
		return err
	}

	// 清除缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:permissions:%s:%s", roleCode, domain))
	}

	return nil
}

// RemovePermission 移除域内权限
func (m *RBACManager) RemovePermission(ctx context.Context, roleCode, domain, resource, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.RemovePolicy(roleCode, domain, resource, action)
	if err != nil {
		return err
	}

	// 清除缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:permissions:%s:%s", roleCode, domain))
	}

	return nil
}

// GetPermissions 获取角色域内权限
func (m *RBACManager) GetPermissions(ctx context.Context, roleCode, domain string) ([]*Permission, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("rbac:permissions:%s:%s", roleCode, domain)
	if m.cache != nil {
		if cached, err := m.cache.Get(cacheKey); err == nil {
			if perms, ok := cached.([]*Permission); ok {
				return perms, nil
			}
		}
	}

	// 从 Casbin 获取
	policies := m.enforcer.GetFilteredPolicy(0, roleCode, domain)
	permissions := make([]*Permission, 0, len(policies))

	for _, policy := range policies {
		if len(policy) >= 4 {
			permissions = append(permissions, &Permission{
				RoleCode: policy[0],
				Domain:   policy[1],
				Resource: policy[2],
				Action:   policy[3],
			})
		}
	}

	// 缓存结果
	if m.cache != nil {
		m.cache.Set(cacheKey, permissions, 300)
	}

	return permissions, nil
}

// AssignRole 在域内分配角色给用户
func (m *RBACManager) AssignRole(ctx context.Context, userUUID, roleCode, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.AddGroupingPolicy(userUUID, roleCode, domain)
	if err != nil {
		return err
	}

	// 清除用户角色缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:user_roles:%s:%s", userUUID, domain))
	}

	return nil
}

// RevokeRole 在域内撤销角色
func (m *RBACManager) RevokeRole(ctx context.Context, userUUID, roleCode, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.RemoveGroupingPolicy(userUUID, roleCode, domain)
	if err != nil {
		return err
	}

	// 清除用户角色缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:user_roles:%s:%s", userUUID, domain))
	}

	return nil
}

// GetUserRoles 获取用户域内角色
func (m *RBACManager) GetUserRoles(ctx context.Context, userUUID, domain string) ([]*Role, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("rbac:user_roles:%s:%s", userUUID, domain)
	if m.cache != nil {
		if cached, err := m.cache.Get(cacheKey); err == nil {
			if roles, ok := cached.([]*Role); ok {
				return roles, nil
			}
		}
	}

	// 从 Casbin 获取
	groupingPolicy := m.enforcer.GetFilteredGroupingPolicy(0, userUUID, "", domain)
	roleMap := make(map[string]bool)
	for _, policy := range groupingPolicy {
		if len(policy) >= 2 {
			roleMap[policy[1]] = true
		}
	}

	roles := make([]*Role, 0, len(roleMap))
	for code := range roleMap {
		roles = append(roles, &Role{
			Code: code,
			Name: code,
		})
	}

	// 缓存结果
	if m.cache != nil {
		m.cache.Set(cacheKey, roles, 300)
	}

	return roles, nil
}

// CheckPermission 域内权限检查
func (m *RBACManager) CheckPermission(ctx context.Context, userUUID, domain, resource, action string) (bool, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("rbac:check:%s:%s:%s:%s", userUUID, domain, resource, action)
	if m.cache != nil {
		if cached, err := m.cache.Get(cacheKey); err == nil {
			if allowed, ok := cached.(bool); ok {
				return allowed, nil
			}
		}
	}

	// 执行权限检查
	allowed, err := m.enforcer.Enforce(userUUID, domain, resource, action)
	if err != nil {
		return false, err
	}

	// 缓存结果
	if m.cache != nil {
		m.cache.Set(cacheKey, allowed, 60)
	}

	return allowed, nil
}

// BatchCheckPermission 批量权限检查
func (m *RBACManager) BatchCheckPermission(ctx context.Context, userUUID string, permissions []Permission) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, perm := range permissions {
		if perm.Domain == "" {
			return nil, fmt.Errorf("domain is required for batch permission check")
		}
		key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
		allowed, err := m.CheckPermission(ctx, userUUID, perm.Domain, perm.Resource, perm.Action)
		if err != nil {
			return nil, err
		}
		results[key] = allowed
	}

	return results, nil
}

// GetPolicy 获取所有策略
func (m *RBACManager) GetPolicy(ctx context.Context) ([]*Policy, error) {
	policies := m.enforcer.GetPolicy()
	result := make([]*Policy, 0, len(policies))

	for _, policy := range policies {
		if len(policy) >= 4 {
			result = append(result, &Policy{
				Subject: policy[0],
				Domain:  policy[1],
				Object:  policy[2],
				Action:  policy[3],
			})
		}
	}

	return result, nil
}

// ClearPolicy 清空所有策略
func (m *RBACManager) ClearPolicy(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enforcer.ClearPolicy()

	// 清除所有缓存
	if m.cache != nil {
		// 这里应该清除所有 rbac 相关的缓存
		// 实际实现需要更复杂的缓存键管理
	}

	return nil
}

// AddPolicy 添加域内策略
func (m *RBACManager) AddPolicy(ctx context.Context, subject, domain, object, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.AddPolicy(subject, domain, object, action)
	return err
}

// RemovePolicy 移除域内策略
func (m *RBACManager) RemovePolicy(ctx context.Context, subject, domain, object, action string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.RemovePolicy(subject, domain, object, action)
	return err
}

// AddPolicies 批量添加策略
func (m *RBACManager) AddPolicies(ctx context.Context, policies [][]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.AddPolicies(policies)
	return err
}

// RemovePolicies 批量移除策略
func (m *RBACManager) RemovePolicies(ctx context.Context, policies [][]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.enforcer.RemovePolicies(policies)
	return err
}

// SavePolicy 保存策略
func (m *RBACManager) SavePolicy() error {
	return m.enforcer.SavePolicy()
}

// LoadPolicy 加载策略
func (m *RBACManager) LoadPolicy() error {
	return m.enforcer.LoadPolicy()
}

// AddRoleInheritance 在域内添加角色继承关系（子角色继承父角色）
func (m *RBACManager) AddRoleInheritance(ctx context.Context, childRoleCode, parentRoleCode, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 在 Casbin 中添加角色继承关系：g, child_role, parent_role, domain
	_, err := m.enforcer.AddGroupingPolicy(childRoleCode, parentRoleCode, domain)
	if err != nil {
		return err
	}

	// 清除相关缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:role_inheritance:%s:%s", childRoleCode, domain))
	}

	return nil
}

// RemoveRoleInheritance 移除角色的域内继承关系
func (m *RBACManager) RemoveRoleInheritance(ctx context.Context, roleCode, domain string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 移除角色作为子角色的所有继承关系
	_, err := m.enforcer.RemoveFilteredGroupingPolicy(0, roleCode, "", domain)
	if err != nil {
		return err
	}

	// 清除相关缓存
	if m.cache != nil {
		m.cache.Delete(fmt.Sprintf("rbac:role_inheritance:%s:%s", roleCode, domain))
	}

	return nil
}

// GetRoleInheritance 获取角色的域内父角色
func (m *RBACManager) GetRoleInheritance(ctx context.Context, roleCode, domain string) ([]string, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("rbac:role_inheritance:%s:%s", roleCode, domain)
	if m.cache != nil {
		if cached, err := m.cache.Get(cacheKey); err == nil {
			if parentRoles, ok := cached.([]string); ok {
				return parentRoles, nil
			}
		}
	}

	// 从 Casbin 获取角色的父角色
	groupingPolicy := m.enforcer.GetFilteredGroupingPolicy(0, roleCode, "", domain)
	parentRoles := make([]string, 0, len(groupingPolicy))

	for _, policy := range groupingPolicy {
		if len(policy) >= 2 {
			parentRoles = append(parentRoles, policy[1])
		}
	}

	// 缓存结果
	if m.cache != nil {
		m.cache.Set(cacheKey, parentRoles, 300)
	}

	return parentRoles, nil
}
