package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServeOpenAPI(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "openapi.yaml")
	err := os.WriteFile(tmp, []byte("openapi: 3.0.0"), 0o644)
	require.NoError(t, err)

	handler := ServeOpenAPI(tmp)
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/yaml", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "openapi: 3.0.0")
}
