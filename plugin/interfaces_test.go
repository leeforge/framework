package plugin

import (
	"context"
	"testing"

	"github.com/go-chi/chi/v5"
)

// testFullPlugin implements all interfaces -- verifies compile-time compliance.
type testFullPlugin struct{}

func (p *testFullPlugin) Name() string                              { return "test-full" }
func (p *testFullPlugin) Version() string                           { return "1.0.0" }
func (p *testFullPlugin) Dependencies() []string                    { return nil }
func (p *testFullPlugin) Enable(context.Context, *AppContext) error { return nil }

func (p *testFullPlugin) Install(context.Context, *AppContext) error   { return nil }
func (p *testFullPlugin) Uninstall(context.Context, *AppContext) error { return nil }
func (p *testFullPlugin) Disable(context.Context, *AppContext) error   { return nil }
func (p *testFullPlugin) RegisterRoutes(chi.Router)                    {}
func (p *testFullPlugin) RegisterMiddlewares(chi.Router)               {}
func (p *testFullPlugin) RegisterModels() []any                        { return nil }
func (p *testFullPlugin) SubscribeEvents(EventBus)                     {}
func (p *testFullPlugin) HealthCheck(context.Context) error            { return nil }
func (p *testFullPlugin) PluginOptions() PluginOptions {
	return PluginOptions{Optional: false, Description: "test"}
}

// Compile-time assertions
var _ Plugin = (*testFullPlugin)(nil)
var _ Installable = (*testFullPlugin)(nil)
var _ Uninstallable = (*testFullPlugin)(nil)
var _ Disableable = (*testFullPlugin)(nil)
var _ RouteProvider = (*testFullPlugin)(nil)
var _ MiddlewareProvider = (*testFullPlugin)(nil)
var _ ModelProvider = (*testFullPlugin)(nil)
var _ EventSubscriber = (*testFullPlugin)(nil)
var _ HealthReporter = (*testFullPlugin)(nil)
var _ Configurable = (*testFullPlugin)(nil)

// testMinimalPlugin implements ONLY the core interface -- proves ISP works.
type testMinimalPlugin struct{}

func (p *testMinimalPlugin) Name() string                              { return "test-minimal" }
func (p *testMinimalPlugin) Version() string                           { return "1.0.0" }
func (p *testMinimalPlugin) Dependencies() []string                    { return nil }
func (p *testMinimalPlugin) Enable(context.Context, *AppContext) error { return nil }

var _ Plugin = (*testMinimalPlugin)(nil)

func TestCapabilityDetection(t *testing.T) {
	full := Plugin(&testFullPlugin{})
	minimal := Plugin(&testMinimalPlugin{})

	// Full plugin has all capabilities
	if _, ok := full.(RouteProvider); !ok {
		t.Error("testFullPlugin should implement RouteProvider")
	}
	if _, ok := full.(EventSubscriber); !ok {
		t.Error("testFullPlugin should implement EventSubscriber")
	}
	if _, ok := full.(Configurable); !ok {
		t.Error("testFullPlugin should implement Configurable")
	}

	// Minimal plugin has no optional capabilities
	if _, ok := minimal.(RouteProvider); ok {
		t.Error("testMinimalPlugin should NOT implement RouteProvider")
	}
	if _, ok := minimal.(EventSubscriber); ok {
		t.Error("testMinimalPlugin should NOT implement EventSubscriber")
	}
	if _, ok := minimal.(Configurable); ok {
		t.Error("testMinimalPlugin should NOT implement Configurable")
	}
}
