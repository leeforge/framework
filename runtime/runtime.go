package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/leeforge/framework/plugin"
	"go.uber.org/zap"
)

// Config holds configuration for creating a new Runtime.
type Config struct {
	Router      chi.Router
	DB          plugin.Database
	Redis       *redis.Client
	Logger      *zap.Logger
	EventBuffer int // default 1024
}

// Runtime manages plugin lifecycle with correct dependency ordering.
type Runtime struct {
	router chi.Router
	db     plugin.Database
	redis  *redis.Client
	logger *zap.Logger

	plugins      map[string]plugin.Plugin
	pluginState  map[string]plugin.PluginState
	pluginErrors map[string]error
	pluginModels map[string][]any
	mu           sync.RWMutex

	bootOrder  []string
	appContext *plugin.AppContext
	eventBus   *eventBus

	shutdownCtx context.Context
	shutdownFn  context.CancelFunc

	healthChecks map[string]func(context.Context) error
}

// NewRuntime creates a new runtime instance.
func NewRuntime(cfg Config) *Runtime {
	if cfg.EventBuffer <= 0 {
		cfg.EventBuffer = 1024
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	shutdownCtx, shutdownFn := context.WithCancel(context.Background())
	bus := NewEventBus(cfg.EventBuffer, cfg.Logger)

	rt := &Runtime{
		router:       cfg.Router,
		db:           cfg.DB,
		redis:        cfg.Redis,
		logger:       cfg.Logger,
		plugins:      make(map[string]plugin.Plugin),
		pluginState:  make(map[string]plugin.PluginState),
		pluginErrors: make(map[string]error),
		pluginModels: make(map[string][]any),
		eventBus:     bus,
		shutdownCtx:  shutdownCtx,
		shutdownFn:   shutdownFn,
		healthChecks: make(map[string]func(context.Context) error),
	}

	rt.appContext = &plugin.AppContext{
		Router:   cfg.Router,
		DB:       cfg.DB,
		Redis:    cfg.Redis,
		Logger:   cfg.Logger,
		Services: plugin.NewServiceRegistry(),
		Config:   plugin.EmptyConfig(),
		Events:   bus,
	}

	return rt
}

// Services returns the plugin service registry for pre-registering core services.
// Must be called before Bootstrap.
func (r *Runtime) Services() *plugin.ServiceRegistry {
	return r.appContext.Services
}

// Register adds a plugin. Must be called before Bootstrap.
func (r *Runtime) Register(p plugin.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	r.pluginState[name] = plugin.StateRegistered
	r.logger.Info("plugin registered", zap.String("name", name), zap.String("version", p.Version()))
	return nil
}

// Bootstrap initializes all plugins in dependency order.
func (r *Runtime) Bootstrap(ctx context.Context) error {
	startTime := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}

	// Phase 1: Resolve dependencies
	order, err := r.resolveDependencies()
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}
	r.bootOrder = order
	r.logger.Info("dependency resolution completed", zap.Strings("order", order))

	// Phase 2: Install (only Installable plugins)
	for _, name := range order {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("bootstrap canceled: %w", err)
		}
		if r.pluginState[name] == plugin.StateFailed {
			continue
		}
		if p, ok := r.plugins[name].(plugin.Installable); ok {
			if err := p.Install(ctx, r.appContext); err != nil {
				if abortErr := r.handlePluginError(name, fmt.Errorf("install failed: %w", err)); abortErr != nil {
					return abortErr
				}
				continue
			}
		}
		r.pluginState[name] = plugin.StateInstalled
	}

	// Phase 3: Collect models (only ModelProvider plugins)
	for _, name := range order {
		if r.pluginState[name] == plugin.StateFailed {
			continue
		}
		if p, ok := r.plugins[name].(plugin.ModelProvider); ok {
			r.pluginModels[name] = p.RegisterModels()
		}
	}

	// Phase 4: Enable (in dependency order)
	for _, name := range order {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("bootstrap canceled: %w", err)
		}
		if r.pluginState[name] == plugin.StateFailed {
			continue
		}

		// Check if any dependency is Failed
		if depErr := r.checkDependenciesHealthy(name); depErr != nil {
			if abortErr := r.handlePluginError(name, depErr); abortErr != nil {
				return abortErr
			}
			continue
		}

		if err := r.plugins[name].Enable(ctx, r.appContext); err != nil {
			if abortErr := r.handlePluginError(name, fmt.Errorf("enable failed: %w", err)); abortErr != nil {
				return abortErr
			}
			continue
		}
		r.pluginState[name] = plugin.StateEnabled
	}

	// Phase 5: Register routes & middleware
	for _, name := range order {
		if r.pluginState[name] != plugin.StateEnabled {
			continue
		}
		if p, ok := r.plugins[name].(plugin.RouteProvider); ok {
			p.RegisterRoutes(r.router)
		}
		if p, ok := r.plugins[name].(plugin.MiddlewareProvider); ok {
			p.RegisterMiddlewares(r.router)
		}
	}

	// Phase 6: Subscribe events
	for _, name := range order {
		if r.pluginState[name] != plugin.StateEnabled {
			continue
		}
		if p, ok := r.plugins[name].(plugin.EventSubscriber); ok {
			p.SubscribeEvents(r.eventBus)
		}
	}

	// Phase 7: Register health checks
	for _, name := range order {
		if r.pluginState[name] != plugin.StateEnabled {
			continue
		}
		if p, ok := r.plugins[name].(plugin.HealthReporter); ok {
			r.healthChecks[name] = p.HealthCheck
		}
	}

	r.logger.Info("bootstrap completed",
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("plugins", len(r.plugins)),
	)
	return nil
}

// Shutdown disables plugins in reverse topological order.
func (r *Runtime) Shutdown(ctx context.Context) error {
	r.shutdownFn()

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Close event bus (drain + wait for in-flight)
	r.eventBus.Close()

	// Disable plugins in REVERSE topological order
	reversed := reverseSlice(r.bootOrder)
	for _, name := range reversed {
		if r.pluginState[name] != plugin.StateEnabled {
			continue
		}
		if p, ok := r.plugins[name].(plugin.Disableable); ok {
			if err := p.Disable(shutdownCtx, r.appContext); err != nil {
				r.logger.Error("plugin disable failed",
					zap.String("plugin", name), zap.Error(err))
			}
		}
		r.pluginState[name] = plugin.StateDisabled
	}

	// Close infrastructure
	if r.db != nil {
		r.db.Close()
	}
	if r.redis != nil {
		r.redis.Close()
	}

	r.logger.Info("shutdown completed")
	return nil
}

// Publish sends an event through the event bus.
func (r *Runtime) Publish(ctx context.Context, event plugin.Event) error {
	return r.eventBus.Publish(ctx, event)
}

// GetPluginState returns the state of a plugin by name.
func (r *Runtime) GetPluginState(name string) (plugin.PluginState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.pluginState[name]
	return state, ok
}

// ListPlugins returns a snapshot of all plugin states.
func (r *Runtime) ListPlugins() map[string]plugin.PluginState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]plugin.PluginState, len(r.pluginState))
	for k, v := range r.pluginState {
		result[k] = v
	}
	return result
}

// BootOrder returns the topological order used during bootstrap.
func (r *Runtime) BootOrder() []string {
	return append([]string{}, r.bootOrder...)
}

// GetPluginModels returns models registered by plugins.
func (r *Runtime) GetPluginModels() map[string][]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]any, len(r.pluginModels))
	for k, v := range r.pluginModels {
		result[k] = append([]any{}, v...)
	}
	return result
}

// --- Internal ---

func (r *Runtime) resolveDependencies() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inDegree := make(map[string]int, len(r.plugins))
	dependents := make(map[string][]string) // dep -> list of plugins that depend on it

	for name := range r.plugins {
		inDegree[name] = 0
	}

	for name, p := range r.plugins {
		for _, dep := range p.Dependencies() {
			if _, exists := r.plugins[dep]; !exists {
				return nil, fmt.Errorf("plugin %q depends on %q which is not registered", name, dep)
			}
			inDegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // deterministic

	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, dep := range dependents[current] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	if len(order) != len(r.plugins) {
		return nil, errors.New("circular dependency detected")
	}

	return order, nil
}

func (r *Runtime) handlePluginError(name string, err error) error {
	r.pluginState[name] = plugin.StateFailed
	r.pluginErrors[name] = err

	opts := r.getPluginOptions(name)
	if opts.Optional {
		r.logger.Warn("optional plugin failed, continuing",
			zap.String("plugin", name), zap.Error(err))
		return nil
	}

	return fmt.Errorf("required plugin %q failed: %w", name, err)
}

func (r *Runtime) getPluginOptions(name string) plugin.PluginOptions {
	if p, ok := r.plugins[name].(plugin.Configurable); ok {
		return p.PluginOptions()
	}
	return plugin.PluginOptions{Optional: false}
}

func (r *Runtime) checkDependenciesHealthy(name string) error {
	for _, dep := range r.plugins[name].Dependencies() {
		if r.pluginState[dep] == plugin.StateFailed {
			return fmt.Errorf("dependency %q is in Failed state", dep)
		}
	}
	return nil
}

func reverseSlice(s []string) []string {
	n := len(s)
	reversed := make([]string, n)
	for i, v := range s {
		reversed[n-1-i] = v
	}
	return reversed
}
