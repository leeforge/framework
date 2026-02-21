package plugin

import (
	"fmt"
	"sort"
	"sync"
)

// ServiceRegistry provides type-safe, namespace-aware service registration and lookup.
// Services are keyed by "pluginName.serviceName" (e.g. "audit.service", "rbac.enforcer").
type ServiceRegistry struct {
	services map[string]any
	mu       sync.RWMutex
}

// NewServiceRegistry creates an empty service registry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]any),
	}
}

// Register stores a service. Returns error if key already exists.
func (sr *ServiceRegistry) Register(key string, svc any) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.services[key]; exists {
		return fmt.Errorf("service %q already registered", key)
	}
	sr.services[key] = svc
	return nil
}

// MustRegister stores a service, panicking on duplicate.
func (sr *ServiceRegistry) MustRegister(key string, svc any) {
	if err := sr.Register(key, svc); err != nil {
		panic(err)
	}
}

// Has returns true if a service is registered under the given key.
func (sr *ServiceRegistry) Has(key string) bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	_, exists := sr.services[key]
	return exists
}

// Keys returns all registered service keys, sorted alphabetically.
func (sr *ServiceRegistry) Keys() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	keys := make([]string, 0, len(sr.services))
	for k := range sr.services {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Resolve retrieves a service with compile-time type safety via generics.
func Resolve[T any](sr *ServiceRegistry, key string) (T, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	var zero T
	svc, exists := sr.services[key]
	if !exists {
		return zero, fmt.Errorf("service %q not found", key)
	}

	typed, ok := svc.(T)
	if !ok {
		return zero, fmt.Errorf("service %q is %T, want %T", key, svc, zero)
	}
	return typed, nil
}

// MustResolve retrieves a service, panicking if not found or wrong type.
func MustResolve[T any](sr *ServiceRegistry, key string) T {
	svc, err := Resolve[T](sr, key)
	if err != nil {
		panic(err)
	}
	return svc
}
