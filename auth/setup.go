package auth

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"
	"github.com/leeforge/framework/auth/abac"
	"github.com/leeforge/framework/auth/casbin"
	"github.com/leeforge/framework/auth/rbac"
	"github.com/leeforge/framework/cache"
	"github.com/leeforge/framework/ent"
	"go.uber.org/zap"
)

// Config frame-core/auth 初始化配置
type Config struct {
	DatabaseURL string      // 数据库连接 URL
	AutoMigrate bool        // 是否自动迁移数据库
	EnableCache bool        // 是否启用缓存
	Logger      *zap.Logger // 日志记录器
}

// AuthCore 认证核心实例
type AuthCore struct {
	EntClient   *ent.Client
	RBACManager *rbac.RBACManager
	ABACManager *abac.ABACManager
	Adapter     *casbin.EntAdapter
	logger      *zap.Logger
}

// Setup 初始化 frame-core/auth
// 包括数据库连接、表迁移、RBAC/ABAC 管理器初始化
func Setup(ctx context.Context, cfg Config) (*AuthCore, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	logger.Info("Initializing frame-core/auth...")

	// 1. 初始化 Ent Client
	client, err := ent.Open(dialect.Postgres, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 2. 运行数据库迁移（如果启用）
	if cfg.AutoMigrate {
		logger.Info("Running frame-core/auth database migrations...")
		if err := client.Schema.Create(ctx); err != nil {
			client.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
		logger.Info("Frame-core/auth database migrations completed")
	}

	// 3. 初始化 Casbin 适配器
	adapter := casbin.NewEntAdapter(client)

	// 4. 初始化缓存适配器（可选）
	var cacheAdapter rbac.CacheAdapter
	if cfg.EnableCache {
		cacheAdapter = cache.NewSimpleAdapter()
		logger.Info("Frame-core/auth cache enabled")
	}

	// 5. 初始化 RBAC 管理器
	rbacManager, err := rbac.NewRBACManager(adapter, cacheAdapter)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize RBAC manager: %w", err)
	}

	// 6. 初始化 ABAC 管理器
	abacManager := abac.NewABACManager(cacheAdapter)

	logger.Info("Frame-core/auth initialized successfully",
		zap.Bool("auto_migrate", cfg.AutoMigrate),
		zap.Bool("cache_enabled", cfg.EnableCache),
	)

	return &AuthCore{
		EntClient:   client,
		RBACManager: rbacManager,
		ABACManager: abacManager,
		Adapter:     adapter,
		logger:      logger,
	}, nil
}

// Close 关闭资源
func (ac *AuthCore) Close() error {
	if ac.EntClient != nil {
		return ac.EntClient.Close()
	}
	return nil
}

// CheckUserPermission 检查用户权限（快捷方法）
func (ac *AuthCore) CheckUserPermission(
	ctx context.Context,
	userUUID string,
	domain string,
	resource string,
	action string,
) (bool, error) {
	return ac.RBACManager.CheckPermission(ctx, userUUID, domain, resource, action)
}

// AssignRoleToUser 分配角色给用户（快捷方法）
func (ac *AuthCore) AssignRoleToUser(ctx context.Context, userUUID string, roleCode string, domain string) error {
	return ac.RBACManager.AssignRole(ctx, userUUID, roleCode, domain)
}

// GetUserRoles 获取用户角色（快捷方法）
func (ac *AuthCore) GetUserRoles(ctx context.Context, userUUID string, domain string) ([]*rbac.Role, error) {
	return ac.RBACManager.GetUserRoles(ctx, userUUID, domain)
}
