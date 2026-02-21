package runtime

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/leeforge/framework/plugin"
	"go.uber.org/zap"
)

// --- Test Helpers ---

type testPlugin struct {
	name      string
	version   string
	deps      []string
	enableFn  func(context.Context, *plugin.AppContext) error
	installFn func(context.Context, *plugin.AppContext) error
	disableFn func(context.Context, *plugin.AppContext) error
	options   *plugin.PluginOptions
}

func (p *testPlugin) Name() string           { return p.name }
func (p *testPlugin) Version() string        { return p.version }
func (p *testPlugin) Dependencies() []string { return p.deps }
func (p *testPlugin) Enable(ctx context.Context, app *plugin.AppContext) error {
	if p.enableFn != nil {
		return p.enableFn(ctx, app)
	}
	return nil
}

// Optional interfaces -- only present on specific test plugins
type testInstallablePlugin struct {
	testPlugin
}

func (p *testInstallablePlugin) Install(ctx context.Context, app *plugin.AppContext) error {
	if p.installFn != nil {
		return p.installFn(ctx, app)
	}
	return nil
}

type testDisableablePlugin struct {
	testPlugin
}

func (p *testDisableablePlugin) Disable(ctx context.Context, app *plugin.AppContext) error {
	if p.disableFn != nil {
		return p.disableFn(ctx, app)
	}
	return nil
}

type testConfigurablePlugin struct {
	testPlugin
}

func (p *testConfigurablePlugin) PluginOptions() plugin.PluginOptions {
	if p.options != nil {
		return *p.options
	}
	return plugin.PluginOptions{}
}

// --- Tests ---

func newTestRuntime() *Runtime {
	return NewRuntime(Config{
		Router:      chi.NewRouter(),
		Logger:      zap.NewNop(),
		EventBuffer: 1024,
	})
}

func TestRuntime_RegisterAndBootstrap(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	p := &testPlugin{name: "basic", version: "1.0.0"}
	if err := rt.Register(p); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	state, ok := rt.GetPluginState("basic")
	if !ok {
		t.Fatal("plugin state not found")
	}
	if state != plugin.StateEnabled {
		t.Errorf("state = %v, want Enabled", state)
	}
}

func TestRuntime_DuplicateRegisterFails(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	rt.Register(&testPlugin{name: "dup"})
	if err := rt.Register(&testPlugin{name: "dup"}); err == nil {
		t.Fatal("duplicate Register should fail")
	}
}

func TestRuntime_DependencyOrder(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	var order []string
	a := &testPlugin{name: "a", enableFn: func(ctx context.Context, app *plugin.AppContext) error {
		order = append(order, "a")
		return nil
	}}
	b := &testPlugin{name: "b", deps: []string{"a"}, enableFn: func(ctx context.Context, app *plugin.AppContext) error {
		order = append(order, "b")
		return nil
	}}

	// Register in reverse order to prove sorting works
	rt.Register(b)
	rt.Register(a)

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Errorf("order = %v, want [a, b]", order)
	}
}

func TestRuntime_CircularDependencyDetected(t *testing.T) {
	rt := newTestRuntime()
	rt.Register(&testPlugin{name: "x", deps: []string{"y"}})
	rt.Register(&testPlugin{name: "y", deps: []string{"x"}})

	if err := rt.Bootstrap(context.Background()); err == nil {
		t.Fatal("should detect circular dependency")
	}
}

func TestRuntime_MissingDependencyDetected(t *testing.T) {
	rt := newTestRuntime()
	rt.Register(&testPlugin{name: "needs-missing", deps: []string{"nonexistent"}})

	if err := rt.Bootstrap(context.Background()); err == nil {
		t.Fatal("should detect missing dependency")
	}
}

func TestRuntime_InstallCalledForInstallableOnly(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	var installed atomic.Bool
	ip := &testInstallablePlugin{
		testPlugin: testPlugin{name: "installable"},
	}
	ip.installFn = func(ctx context.Context, app *plugin.AppContext) error {
		installed.Store(true)
		return nil
	}

	plain := &testPlugin{name: "plain"}

	rt.Register(ip)
	rt.Register(plain)

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	if !installed.Load() {
		t.Error("Install should have been called on installable plugin")
	}
}

func TestRuntime_ShutdownReverseOrder(t *testing.T) {
	rt := newTestRuntime()

	var disableOrder []string

	a := &testDisableablePlugin{testPlugin: testPlugin{name: "a"}}
	a.disableFn = func(ctx context.Context, app *plugin.AppContext) error {
		disableOrder = append(disableOrder, "a")
		return nil
	}

	b := &testDisableablePlugin{testPlugin: testPlugin{name: "b", deps: []string{"a"}}}
	b.disableFn = func(ctx context.Context, app *plugin.AppContext) error {
		disableOrder = append(disableOrder, "b")
		return nil
	}

	rt.Register(a)
	rt.Register(b)

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	rt.Shutdown(context.Background())

	// b depends on a, so b must be disabled FIRST (reverse order)
	if len(disableOrder) != 2 || disableOrder[0] != "b" || disableOrder[1] != "a" {
		t.Errorf("disable order = %v, want [b, a]", disableOrder)
	}
}

func TestRuntime_OptionalPluginFailure(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	failing := &testConfigurablePlugin{
		testPlugin: testPlugin{
			name: "optional-fail",
			enableFn: func(ctx context.Context, app *plugin.AppContext) error {
				return fmt.Errorf("intentional failure")
			},
		},
	}
	failing.options = &plugin.PluginOptions{Optional: true}

	ok := &testPlugin{name: "ok-plugin"}

	rt.Register(failing)
	rt.Register(ok)

	// Should NOT fail -- optional plugin failure is tolerated
	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap should succeed with optional plugin failure: %v", err)
	}

	state, _ := rt.GetPluginState("optional-fail")
	if state != plugin.StateFailed {
		t.Errorf("optional-fail state = %v, want Failed", state)
	}

	state, _ = rt.GetPluginState("ok-plugin")
	if state != plugin.StateEnabled {
		t.Errorf("ok-plugin state = %v, want Enabled", state)
	}
}

func TestRuntime_RequiredPluginFailureAbortsBootstrap(t *testing.T) {
	rt := newTestRuntime()

	failing := &testPlugin{
		name: "required-fail",
		enableFn: func(ctx context.Context, app *plugin.AppContext) error {
			return fmt.Errorf("critical failure")
		},
	}

	rt.Register(failing)

	if err := rt.Bootstrap(context.Background()); err == nil {
		t.Fatal("Bootstrap should fail when required plugin fails")
	}
}

func TestRuntime_ListPlugins(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	rt.Register(&testPlugin{name: "alpha"})
	rt.Register(&testPlugin{name: "beta"})

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	plugins := rt.ListPlugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
}

func TestRuntime_BootOrder(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	rt.Register(&testPlugin{name: "c", deps: []string{"b"}})
	rt.Register(&testPlugin{name: "a"})
	rt.Register(&testPlugin{name: "b", deps: []string{"a"}})

	if err := rt.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	order := rt.BootOrder()
	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("boot order = %v, want [a, b, c]", order)
	}
}

func TestRuntime_EventsIntegration(t *testing.T) {
	rt := newTestRuntime()
	defer rt.Shutdown(context.Background())

	var received atomic.Bool
	p := &testPlugin{
		name: "eventer",
		enableFn: func(ctx context.Context, app *plugin.AppContext) error {
			app.Events.Subscribe("test.ping", func(ctx context.Context, e plugin.Event) error {
				received.Store(true)
				return nil
			})
			return nil
		},
	}

	rt.Register(p)
	rt.Bootstrap(context.Background())

	rt.Publish(context.Background(), plugin.Event{Name: "test.ping"})
	time.Sleep(100 * time.Millisecond)

	if !received.Load() {
		t.Error("event handler should have been called")
	}
}
