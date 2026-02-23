package migration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeStrategy struct{ called bool }

func (f *fakeStrategy) Name() string { return "fake" }

func (f *fakeStrategy) Migrate(context.Context) error {
	f.called = true
	return nil
}

func TestManager_RunCallsStrategy(t *testing.T) {
	s := &fakeStrategy{}
	m := NewManager(s)
	require.NoError(t, m.Run(context.Background()))
	require.True(t, s.called)
}
