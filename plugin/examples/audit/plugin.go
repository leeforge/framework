package audit

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leeforge/framework/plugin"
	"go.uber.org/zap"
)

// AuditPlugin records security-relevant system events.
//
// Implements: Plugin, Installable, Disableable, RouteProvider, EventSubscriber, HealthReporter, Configurable
type AuditPlugin struct {
	service *AuditService
	logger  *zap.Logger
}

// --- Core Interface (mandatory) ---

func (p *AuditPlugin) Name() string           { return "audit" }
func (p *AuditPlugin) Version() string        { return "1.0.0" }
func (p *AuditPlugin) Dependencies() []string { return []string{"rbac"} }

func (p *AuditPlugin) Enable(ctx context.Context, app *plugin.AppContext) error {
	p.logger = app.Logger
	p.service = NewAuditService(app.Logger)

	retentionDays := app.Config.GetInt("retention_days", 90)
	p.service.SetRetention(retentionDays)

	return app.Services.Register("audit.service", p.service)
}

// --- Installable ---

func (p *AuditPlugin) Install(ctx context.Context, app *plugin.AppContext) error {
	app.Logger.Info("audit plugin: running DB migrations")
	return nil
}

// --- Disableable ---

func (p *AuditPlugin) Disable(ctx context.Context, app *plugin.AppContext) error {
	app.Logger.Info("audit plugin: flushing pending logs")
	return nil
}

// --- RouteProvider ---

func (p *AuditPlugin) RegisterRoutes(router chi.Router) {
	router.Route("/api/v1/audit", func(r chi.Router) {
		r.Get("/logs", p.handleGetLogs)
		r.Post("/clear", p.handleClearLogs)
	})
}

// --- EventSubscriber ---

func (p *AuditPlugin) SubscribeEvents(bus plugin.EventBus) {
	topics := []string{
		"user.created", "user.updated", "user.deleted",
		"permission.changed", "role.assigned", "role.revoked",
	}
	for _, topic := range topics {
		topic := topic
		bus.Subscribe(topic, func(ctx context.Context, e plugin.Event) error {
			p.service.Record(topic, e.Data)
			return nil
		})
	}
}

// --- HealthReporter ---

func (p *AuditPlugin) HealthCheck(ctx context.Context) error {
	if p.service == nil {
		return fmt.Errorf("audit service not initialized")
	}
	return nil
}

// --- Configurable ---

func (p *AuditPlugin) PluginOptions() plugin.PluginOptions {
	return plugin.PluginOptions{
		Description: "System audit logging for security events",
	}
}

// --- Compile-time interface checks ---

var (
	_ plugin.Plugin          = (*AuditPlugin)(nil)
	_ plugin.Installable     = (*AuditPlugin)(nil)
	_ plugin.Disableable     = (*AuditPlugin)(nil)
	_ plugin.RouteProvider   = (*AuditPlugin)(nil)
	_ plugin.EventSubscriber = (*AuditPlugin)(nil)
	_ plugin.HealthReporter  = (*AuditPlugin)(nil)
	_ plugin.Configurable    = (*AuditPlugin)(nil)
)

// --- Internal ---

// AuditService handles audit log storage and retrieval.
type AuditService struct {
	logger    *zap.Logger
	retention int
}

func NewAuditService(logger *zap.Logger) *AuditService {
	return &AuditService{logger: logger, retention: 90}
}

func (s *AuditService) SetRetention(days int) { s.retention = days }

func (s *AuditService) Record(action string, data any) {
	s.logger.Info("audit record", zap.String("action", action), zap.Any("data", data))
}

func (p *AuditPlugin) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"logs":[]}`))
}

func (p *AuditPlugin) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
