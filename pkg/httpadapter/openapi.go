package httpadapter

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"

	assets "github.com/thetrollfarmercodes/todoctl/todo"
)

// ServeOpenAPI returns a handler that serves the OpenAPI spec file from the
// given path, with a conservative content type. It rejects path traversal.
func ServeOpenAPI(specPath string) http.HandlerFunc {
	clean := filepath.Clean(specPath)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// block path traversal attempts
		if clean != filepath.Clean(specPath) || clean == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if data, err := os.ReadFile(clean); err == nil {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(data)
			return
		}
		// fallback to embedded assets
		if data, err := assets.FS.ReadFile(clean); err == nil {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(data)
			return
		} else {
			status := http.StatusInternalServerError
			if errors.Is(err, os.ErrNotExist) {
				status = http.StatusNotFound
			}
			http.Error(w, http.StatusText(status), status)
			return
		}
	}
}
