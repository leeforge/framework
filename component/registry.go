package component

import (
	"fmt"
	"sync"
)

var (
	registry = &ComponentRegistry{
		components: make(map[string]Component),
	}
)

// ComponentRegistry 组件注册中心
type ComponentRegistry struct {
	mu         sync.RWMutex
	components map[string]Component
}

// Register 注册组件
func Register(component Component) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	name := component.Name()
	if name == "" {
		return fmt.Errorf("component name cannot be empty")
	}

	if _, exists := registry.components[name]; exists {
		return fmt.Errorf("component %s already registered", name)
	}

	registry.components[name] = component
	return nil
}

// Get 获取组件
func Get(name string) (Component, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	component, exists := registry.components[name]
	if !exists {
		return nil, fmt.Errorf("component %s not found", name)
	}

	return component, nil
}

// MustGet 获取组件（panic if not found）
func MustGet(name string) Component {
	component, err := Get(name)
	if err != nil {
		panic(err)
	}
	return component
}

// List 列出所有已注册组件
func List() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.components))
	for name := range registry.components {
		names = append(names, name)
	}
	return names
}

// Clear 清空注册表（仅用于测试）
func Clear() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.components = make(map[string]Component)
}
