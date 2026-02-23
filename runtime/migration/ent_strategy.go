package migration

import "context"

// EntStrategy adapts an ent migration function to Strategy.
type EntStrategy struct {
	migrateFn func(context.Context) error
}

func NewEntStrategy(migrateFn func(context.Context) error) *EntStrategy {
	return &EntStrategy{migrateFn: migrateFn}
}

func (s *EntStrategy) Name() string {
	return "ent"
}

func (s *EntStrategy) Migrate(ctx context.Context) error {
	if s == nil || s.migrateFn == nil {
		return nil
	}
	return s.migrateFn(ctx)
}
