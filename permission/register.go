package permission

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Register registers a route with metadata using an explicit method.
func Register(r chi.Router, method, path string, handler http.Handler, meta Meta) {
	if r == nil || handler == nil {
		return
	}
	r.Method(method, path, Wrap(handler, meta))
}

// Get registers a GET route with metadata.
func Get(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodGet, path, handler, meta)
}

// Post registers a POST route with metadata.
func Post(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodPost, path, handler, meta)
}

// Put registers a PUT route with metadata.
func Put(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodPut, path, handler, meta)
}

// Delete registers a DELETE route with metadata.
func Delete(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodDelete, path, handler, meta)
}

// Patch registers a PATCH route with metadata.
func Patch(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodPatch, path, handler, meta)
}

// Options registers an OPTIONS route with metadata.
func Options(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodOptions, path, handler, meta)
}

// Head registers a HEAD route with metadata.
func Head(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodHead, path, handler, meta)
}

// Trace registers a TRACE route with metadata.
func Trace(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodTrace, path, handler, meta)
}

// Connect registers a CONNECT route with metadata.
func Connect(r chi.Router, path string, handler http.HandlerFunc, meta Meta) {
	Register(r, http.MethodConnect, path, handler, meta)
}
