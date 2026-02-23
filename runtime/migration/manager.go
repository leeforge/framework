package migration

import "context"

// Manager orchestrates migration execution.
type Manager struct {
	strategy Strategy
}

func NewManager(strategy Strategy) *Manager {
	return &Manager{strategy: strategy}
}

func (m *Manager) Run(ctx context.Context) error {
	if m == nil || m.strategy == nil {
		return nil
	}
	return m.strategy.Migrate(ctx)
}
