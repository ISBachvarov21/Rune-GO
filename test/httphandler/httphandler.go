package httphandler

import (
	"net/http"
)

type HTTPHandler struct {
	routes map[string]map[string]http.HandlerFunc
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// search for handler inside route->method in routes map
	methods, ok := h.routes[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	handler, ok := methods[r.Method]
	if !ok {
		http.NotFound(w, r)
		return
	}
	handler(w, r)

	// todo: if not a match try searching for a match in dynamic paths
}

func (h *HTTPHandler) AddRoute(method string, path string, handler http.HandlerFunc) {
	if h.routes == nil {
		h.routes = map[string]map[string]http.HandlerFunc{}
	}
	if h.routes[path] == nil {
		h.routes[path] = map[string]http.HandlerFunc{}
	}
	h.routes[path][method] = handler
}
