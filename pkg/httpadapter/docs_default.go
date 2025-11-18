//go:build !fuego

package httpadapter

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// attachDocs mounts the OpenAPI spec using the local file server.
func attachDocs(r chi.Router, cfg *HTTPConfig) {
	if cfg == nil || cfg.OpenAPIPath == "" {
		return
	}
	openapiPath := filepath.Clean(cfg.OpenAPIPath)
	handler := ServeOpenAPI(openapiPath)
	r.Method(http.MethodGet, "/openapi.yaml", handler)
	r.Method(http.MethodHead, "/openapi.yaml", handler)
}
