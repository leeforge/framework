package migration

import "context"

// Strategy defines a pluggable migration execution strategy.
type Strategy interface {
	Name() string
	Migrate(ctx context.Context) error
}
