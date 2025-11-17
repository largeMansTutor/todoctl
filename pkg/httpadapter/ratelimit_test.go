package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name      string
		setupReqs func() []*http.Request
		wantCodes []int
	}{
		{
			name: "allows then blocks same key",
			setupReqs: func() []*http.Request {
				req := httptest.NewRequest(http.MethodGet, "/v1/todos", nil)
				req.RemoteAddr = "1.2.3.4:1234"
				return []*http.Request{req, req}
			},
			wantCodes: []int{http.StatusOK, http.StatusTooManyRequests},
		},
		{
			name: "separate buckets per api key",
			setupReqs: func() []*http.Request {
				reqA := httptest.NewRequest(http.MethodGet, "/v1/todos", nil)
				reqA.RemoteAddr = "1.1.1.1:1111"
				reqA.Header.Set("X-API-Key", "keyA")
				reqB := httptest.NewRequest(http.MethodGet, "/v1/todos", nil)
				reqB.RemoteAddr = "1.1.1.1:1111"
				reqB.Header.Set("X-API-Key", "keyB")
				return []*http.Request{reqA, reqB, reqA}
			},
			wantCodes: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(1, 1, nil)
			handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			reqs := tt.setupReqs()
			for i, req := range reqs {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				require.Equal(t, tt.wantCodes[i], rec.Code)
			}
		})
	}
}

func TestRateLimiter_NoopWhenNil(t *testing.T) {
	var rl *RateLimiter
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestClientIPPrefersXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 192.168.0.1")

	ip := clientIP(req)
	require.Equal(t, "203.0.113.5", ip)
}
