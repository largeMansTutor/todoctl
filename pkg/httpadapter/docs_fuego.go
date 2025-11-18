//go:build fuego

package httpadapter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-fuego/fuego"
)

// attachDocs uses fuego helpers to serve the OpenAPI spec and UI.
func attachDocs(r chi.Router, cfg *HTTPConfig) {
	if cfg == nil || cfg.OpenAPIPath == "" {
		return
	}
	specRoute := "/openapi.yaml"
	specHandler := ServeOpenAPI(cfg.OpenAPIPath)
	r.Method(http.MethodGet, specRoute, specHandler)
	r.Method(http.MethodHead, specRoute, specHandler)
	uiHandler := fuego.DefaultOpenAPIHandler(specRoute)
	r.Handle("/docs", uiHandler)
	r.Handle("/docs/", uiHandler)
}
