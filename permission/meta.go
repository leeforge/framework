package permission

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Meta holds permission codes and route metadata for a handler.
type Meta struct {
	Description string
	IsPublic    bool
	Permissions []string
}

// MetaHandler wraps a handler with permission metadata.
type MetaHandler struct {
	handler http.Handler
	Meta    Meta
}

func (h *MetaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

// Wrap attaches metadata to a handler.
func Wrap(handler http.Handler, meta Meta) http.Handler {
	if handler == nil {
		return handler
	}
	return &MetaHandler{handler: handler, Meta: meta}
}

// ExtractMeta returns metadata from a handler if present.
func ExtractMeta(handler http.Handler) (Meta, bool) {
	for handler != nil {
		if wrapped, ok := handler.(*MetaHandler); ok {
			return wrapped.Meta, true
		}
		// chi wraps handlers with ChainHandler when middleware is applied.
		// Unwrap to reach the underlying endpoint.
		if chained, ok := handler.(*chi.ChainHandler); ok {
			handler = chained.Endpoint
			continue
		}
		break
	}
	return Meta{}, false
}

// Public creates metadata for a public route.
func Public(description string, codes ...string) Meta {
	return Meta{
		Description: description,
		IsPublic:    true,
		Permissions: codes,
	}
}

// Private creates metadata for a protected route.
func Private(description string, codes ...string) Meta {
	return Meta{
		Description: description,
		IsPublic:    false,
		Permissions: codes,
	}
}
