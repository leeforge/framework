package plugin

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Database is the minimal interface for DB lifecycle management.
// Keeps frame-core decoupled from specific generated Ent clients.
type Database interface {
	Close() error
}

// AppContext is the typed dependency injection context passed to all plugin lifecycle methods.
type AppContext struct {
	Router   chi.Router
	DB       Database
	Redis    *redis.Client
	Logger   *zap.Logger
	Services *ServiceRegistry
	Config   ConfigProvider
	Events   EventBus
}
