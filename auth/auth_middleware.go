package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// AuthConfig 认证配置
type AuthConfig struct {
	RequireJWT       bool     // 是否需要 JWT
	RequireAPIKey    bool     // 是否需要 API Key
	EnableDataFilter bool     // 是否启用数据过滤
	AllowedAPIKeys   []string // 允许的 API Key (用于测试)
}

// APIKeyInfo API Key 信息
type APIKeyInfo struct {
	Key         string                 `json:"key"`
	CreatedBy   string                 `json:"created_by"`
	ExpiredAt   time.Time              `json:"expired_at"`
	Permissions []Permission           `json:"permissions"`
	DataFilters map[string]interface{} `json:"data_filters"`
	RateLimit   RateLimitConfig        `json:"rate_limit"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Minute int `json:"minute"`
	Daily  int `json:"daily"`
	Burst  int `json:"burst"`
}

// Permission 权限
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	config      AuthConfig
	apiKeyStore APIKeyStore
	jwtSecret   string
	logger      *zap.Logger
}

// APIKeyStore API Key 存储接口
type APIKeyStore interface {
	GetByKey(ctx context.Context, key string) (*APIKeyInfo, error)
	Validate(ctx context.Context, key string) error
}

// JWTValidator JWT 验证器接口
type JWTValidator interface {
	Validate(token string) (string, error) // 返回 userID
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(config AuthConfig, store APIKeyStore, jwtSecret string, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		config:      config,
		apiKeyStore: store,
		jwtSecret:   jwtSecret,
		logger:      logger,
	}
}

// Middleware 认证中间件
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. API Key 验证 (必需)
		apiKey := r.Header.Get("X-API-Key")
		if a.config.RequireAPIKey && apiKey == "" {
			a.writeError(w, 401, 4006, "API-Key is required")
			return
		}

		// 2. 验证 API Key
		var keyInfo *APIKeyInfo
		if apiKey != "" {
			var err error
			keyInfo, err = a.validateAPIKey(r.Context(), apiKey)
			if err != nil {
				a.writeError(w, 401, 4006, "Invalid API-Key")
				return
			}

			// 3. 检查过期和约束
			if err := a.checkKeyConstraints(keyInfo); err != nil {
				a.writeError(w, 401, 4006, err.Error())
				return
			}
		}

		// 4. JWT 验证 (可选，仅需要用户身份时)
		var userID string
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			jwtToken := strings.TrimPrefix(authHeader, "Bearer ")
			userID, err := a.validateJWT(jwtToken)
			if err != nil {
				a.writeError(w, 401, 4006, "Invalid JWT token")
				return
			}

			// 5. 验证用户 ID 与 API Key 创建者一致
			if keyInfo != nil && userID != keyInfo.CreatedBy {
				a.writeError(w, 403, 4005, "User mismatch with API-Key")
				return
			}
		}

		// 6. 存入 Context
		ctx := r.Context()
		if keyInfo != nil {
			ctx = context.WithValue(ctx, "api_key_info", keyInfo)
		}
		if userID != "" {
			ctx = context.WithValue(ctx, "user_id", userID)
		}

		// 7. 应用数据过滤
		if keyInfo != nil && a.config.EnableDataFilter {
			ctx = context.WithValue(ctx, "data_filters", keyInfo.DataFilters)
		}

		// 8. 添加请求追踪
		traceID := generateTraceID()
		ctx = context.WithValue(ctx, "trace_id", traceID)

		// 9. 继续处理请求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateAPIKey 验证 API Key
func (a *AuthMiddleware) validateAPIKey(ctx context.Context, apiKey string) (*APIKeyInfo, error) {
	// 测试模式
	if a.config.AllowedAPIKeys != nil {
		for _, allowed := range a.config.AllowedAPIKeys {
			if apiKey == allowed {
				return &APIKeyInfo{
					Key:       apiKey,
					CreatedBy: "test-user",
					ExpiredAt: time.Now().Add(24 * time.Hour),
					DataFilters: map[string]interface{}{
						"tenant_id": "test-tenant",
					},
					RateLimit: RateLimitConfig{
						Minute: 100,
						Daily:  1000,
						Burst:  10,
					},
				}, nil
			}
		}
	}

	// 生产模式：从存储验证
	if a.apiKeyStore != nil {
		return a.apiKeyStore.GetByKey(ctx, apiKey)
	}

	return nil, fmt.Errorf("no API key store configured")
}

// checkKeyConstraints 检查 API Key 约束
func (a *AuthMiddleware) checkKeyConstraints(keyInfo *APIKeyInfo) error {
	// 检查过期
	if time.Now().After(keyInfo.ExpiredAt) {
		return fmt.Errorf("API key expired")
	}

	// 检查权限
	if len(keyInfo.Permissions) == 0 {
		return fmt.Errorf("API key has no permissions")
	}

	return nil
}

// validateJWT 验证 JWT
func (a *AuthMiddleware) validateJWT(token string) (string, error) {
	// 简化实现
	// 实际应该使用 JWT 库验证
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// 模拟验证
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// 这里应该验证签名和过期
	// 返回 userID
	return "user-123", nil
}

// writeError 写入错误响应
func (a *AuthMiddleware) writeError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%d,"message":"%s"}}`, code, message)
}

// generateTraceID 生成追踪 ID
func generateTraceID() string {
	// 简化实现
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

// GetUserInfoFromContext 从上下文获取用户信息
func GetUserInfoFromContext(ctx context.Context) (userID string, keyInfo *APIKeyInfo, filters map[string]interface{}) {
	if uid, ok := ctx.Value("user_id").(string); ok {
		userID = uid
	}
	if info, ok := ctx.Value("api_key_info").(*APIKeyInfo); ok {
		keyInfo = info
	}
	if f, ok := ctx.Value("data_filters").(map[string]interface{}); ok {
		filters = f
	}
	return
}

// AuthMiddlewareChain 认证中间件链
func AuthMiddlewareChain(config AuthConfig, store APIKeyStore, jwtSecret string, logger *zap.Logger) func(next http.Handler) http.Handler {
	auth := NewAuthMiddleware(config, store, jwtSecret, logger)
	return auth.Middleware
}

// RBACMiddleware RBAC 中间件
type RBACMiddleware struct {
	rbacManager RBACManager
	logger      *zap.Logger
}

// RBACManager RBAC 管理器接口
type RBACManager interface {
	CheckPermission(ctx context.Context, userUUID, domain, resource, action string) (bool, error)
}

// NewRBACMiddleware 创建 RBAC 中间件
func NewRBACMiddleware(rbacManager RBACManager, logger *zap.Logger) *RBACMiddleware {
	return &RBACMiddleware{
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// Middleware RBAC 中间件
func (r *RBACMiddleware) Middleware(domain string, resource string, action string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			// 0. Super Admin Bypass
			if isSuper, ok := ctx.Value("is_super_admin").(bool); ok && isSuper {
				next.ServeHTTP(w, req)
				return
			}

			// 获取用户 ID
			userID, _, _ := GetUserInfoFromContext(ctx)
			if userID == "" {
				// 如果没有用户 ID，检查是否允许匿名访问
				// 这里可以根据需要调整
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// 检查权限
			allowed, err := r.rbacManager.CheckPermission(ctx, userID, domain, resource, action)
			if err != nil {
				r.logger.Error("RBAC check failed", zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// ABACMiddleware ABAC 中间件
type ABACMiddleware struct {
	abacManager ABACManager
	logger      *zap.Logger
}

// ABACManager ABAC 管理器接口
type ABACManager interface {
	CheckPermission(
		ctx context.Context,
		userAttrs map[string]interface{},
		resourceAttrs map[string]interface{},
		action string,
		contextAttrs map[string]interface{},
	) (bool, error)
}

// NewABACMiddleware 创建 ABAC 中间件
func NewABACMiddleware(abacManager ABACManager, logger *zap.Logger) *ABACMiddleware {
	return &ABACMiddleware{
		abacManager: abacManager,
		logger:      logger,
	}
}

// Middleware ABAC 中间件
func (a *ABACMiddleware) Middleware(resource string, action string, resourceAttrs map[string]interface{}) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			// 0. Super Admin Bypass
			if isSuper, ok := ctx.Value("is_super_admin").(bool); ok && isSuper {
				next.ServeHTTP(w, req)
				return
			}

			// 获取用户属性
			userID, _, _ := GetUserInfoFromContext(ctx)
			userAttrs := map[string]interface{}{
				"id": userID,
			}

			// 获取上下文属性
			contextAttrs := map[string]interface{}{
				"ip":         req.RemoteAddr,
				"user_agent": req.UserAgent(),
				"time":       time.Now(),
			}

			// 检查权限
			allowed, err := a.abacManager.CheckPermission(ctx, userAttrs, resourceAttrs, action, contextAttrs)
			if err != nil {
				a.logger.Error("ABAC check failed", zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// DataFilterMiddleware 数据过滤中间件
type DataFilterMiddleware struct {
	logger *zap.Logger
}

// NewDataFilterMiddleware 创建数据过滤中间件
func NewDataFilterMiddleware(logger *zap.Logger) *DataFilterMiddleware {
	return &DataFilterMiddleware{
		logger: logger,
	}
}

// Middleware 数据过滤中间件
func (d *DataFilterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// 获取数据过滤规则
		_, _, filters := GetUserInfoFromContext(ctx)

		if len(filters) > 0 {
			// 将过滤规则注入 context
			ctx = context.WithValue(ctx, "data_filters", filters)
		}

		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

// UnifiedAuthMiddleware 统一认证中间件
type UnifiedAuthMiddleware struct {
	config      AuthConfig
	apiKeyStore APIKeyStore
	rbacManager RBACManager
	abacManager ABACManager
	logger      *zap.Logger
}

// NewUnifiedAuthMiddleware 创建统一认证中间件
func NewUnifiedAuthMiddleware(
	config AuthConfig,
	apiKeyStore APIKeyStore,
	rbacManager RBACManager,
	abacManager ABACManager,
	logger *zap.Logger,
) *UnifiedAuthMiddleware {
	return &UnifiedAuthMiddleware{
		config:      config,
		apiKeyStore: apiKeyStore,
		rbacManager: rbacManager,
		abacManager: abacManager,
		logger:      logger,
	}
}

// Middleware 统一认证中间件
func (u *UnifiedAuthMiddleware) Middleware(next http.Handler) http.Handler {
	// 组合多个中间件
	chain := AuthMiddlewareChain(u.config, u.apiKeyStore, "", u.logger)

	return chain(next)
}

// WithRBAC 添加 RBAC 检查
func (u *UnifiedAuthMiddleware) WithRBAC(resource, action string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		rbac := NewRBACMiddleware(u.rbacManager, u.logger)
		// Default to platform domain for standalone middleware composition.
		return rbac.Middleware("platform", resource, action)(next)
	}
}

// WithABAC 添加 ABAC 检查
func (u *UnifiedAuthMiddleware) WithABAC(resource, action string, resourceAttrs map[string]interface{}) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		abac := NewABACMiddleware(u.abacManager, u.logger)
		return abac.Middleware(resource, action, resourceAttrs)(next)
	}
}

// WithDataFilter 添加数据过滤
func (u *UnifiedAuthMiddleware) WithDataFilter() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		filter := NewDataFilterMiddleware(u.logger)
		return filter.Middleware(next)
	}
}

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(router chi.Router, authMiddleware *UnifiedAuthMiddleware) {
	router.Route("/auth", func(r chi.Router) {
		// API Key 管理
		r.Post("/api-keys", http.HandlerFunc(authMiddleware.WithRBAC("api_key", "create")(http.HandlerFunc(createAPIKey)).ServeHTTP))
		r.Get("/api-keys", http.HandlerFunc(authMiddleware.WithRBAC("api_key", "read")(http.HandlerFunc(listAPIKeys)).ServeHTTP))
		r.Delete("/api-keys/{id}", http.HandlerFunc(authMiddleware.WithRBAC("api_key", "delete")(http.HandlerFunc(deleteAPIKey)).ServeHTTP))

		// 权限检查
		r.Post("/check", http.HandlerFunc(authMiddleware.Middleware(http.HandlerFunc(checkPermission)).ServeHTTP))
	})
}

// createAPIKey 创建 API Key 处理器
func createAPIKey(w http.ResponseWriter, r *http.Request) {
	// 实现创建 API Key 逻辑
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"API key created"}`))
}

// listAPIKeys 列出 API Key 处理器
func listAPIKeys(w http.ResponseWriter, r *http.Request) {
	// 实现列出 API Key 逻辑
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"keys":[]}`))
}

// deleteAPIKey 删除 API Key 处理器
func deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// 实现删除 API Key 逻辑
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"API key deleted"}`))
}

// checkPermission 检查权限处理器
func checkPermission(w http.ResponseWriter, r *http.Request) {
	// 实现权限检查逻辑
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"allowed":true}`))
}
