package plugin

import (
	"context"

	"github.com/go-chi/chi/v5"
)

// Plugin is the minimal interface every plugin must implement.
type Plugin interface {
	Name() string
	Version() string
	Dependencies() []string
	Enable(ctx context.Context, app *AppContext) error
}

// --- Optional Capability Interfaces ---
// Runtime detects these via type assertion: if p, ok := plugin.(RouteProvider); ok { ... }

// Installable -- first-time setup (DB migrations, seed data).
type Installable interface {
	Install(ctx context.Context, app *AppContext) error
}

// Uninstallable -- permanent removal (drop tables, delete files).
type Uninstallable interface {
	Uninstall(ctx context.Context, app *AppContext) error
}

// Disableable -- cleanup on shutdown (release resources, flush buffers).
type Disableable interface {
	Disable(ctx context.Context, app *AppContext) error
}

// RouteProvider -- register HTTP routes.
type RouteProvider interface {
	RegisterRoutes(router chi.Router)
}

// MiddlewareProvider -- register HTTP middleware.
type MiddlewareProvider interface {
	RegisterMiddlewares(router chi.Router)
}

// ModelProvider -- declare Ent ORM models for auto-migration.
type ModelProvider interface {
	RegisterModels() []any
}

// EventSubscriber -- subscribe to system/plugin events.
type EventSubscriber interface {
	SubscribeEvents(bus EventBus)
}

// HealthReporter -- provide custom health checks.
type HealthReporter interface {
	HealthCheck(ctx context.Context) error
}

// Configurable -- declare plugin options (optional flag, description).
type Configurable interface {
	PluginOptions() PluginOptions
}

// PluginOptions holds declarative metadata about a plugin.
type PluginOptions struct {
	Optional    bool   // If true, failure does not abort bootstrap.
	Description string // Human-readable description.
}
