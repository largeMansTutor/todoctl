package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMaxBytes(t *testing.T) {
	handler := MaxBytes(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = r.Body.Read(make([]byte, 10))
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestAPIKeyAuth(t *testing.T) {
	authz := &stubAuth{allowed: map[string]bool{"secret": true}}
	mw := APIKeyAuth(authz, zap.NewNop())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-API-Key", "secret")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code)
}

type stubAuth struct {
	allowed map[string]bool
}

func (s *stubAuth) Authorize(key string, _ ...string) (string, bool) {
	return "", s.allowed[key]
}
