package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTodosFromFlags(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		assertion func(t *testing.T, todos []map[string]any, err error)
	}{
		{
			name:  "parses full fields",
			input: []string{"title|desc|" + time.Now().UTC().Format(time.RFC3339)},
			assertion: func(t *testing.T, todos []map[string]any, err error) {
				require.NoError(t, err)
				require.Len(t, todos, 1)
				assert.Equal(t, "title", todos[0]["title"])
				assert.Equal(t, "desc", todos[0]["description"])
				assert.NotEmpty(t, todos[0]["due_date"])
			},
		},
		{
			name:  "rejects missing title",
			input: []string{"|desc|"},
			assertion: func(t *testing.T, todos []map[string]any, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "rejects bad date",
			input: []string{"title||bad-date"},
			assertion: func(t *testing.T, todos []map[string]any, err error) {
				require.Error(t, err)
			},
		},
		{
			name:  "requires at least one item",
			input: nil,
			assertion: func(t *testing.T, todos []map[string]any, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			todos, err := buildTodosFromFlags(tt.input)
			tt.assertion(t, todos, err)
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	t.Setenv("APP_API_KEY", "env-key")
	assert.Equal(t, "flag-key", resolveAPIKey("flag-key"))
	assert.Equal(t, "env-key", resolveAPIKey(""))
}

func TestDefaultAPIAddr(t *testing.T) {
	assert.Equal(t, "http://localhost:8080", defaultAPIAddr())
	t.Setenv("APP_API_ADDR", "https://example")
	assert.Equal(t, "https://example", defaultAPIAddr())
}

func TestDoAPIRequest(t *testing.T) {
	var received struct {
		method string
		path   string
		body   string
		header http.Header
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.method = r.Method
		received.path = r.URL.Path
		received.header = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		received.body = string(b)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ok":true}`))
	})

	origClient := newHTTPClient
	newHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				return rr.Result(), nil
			}),
		}
	}
	t.Cleanup(func() { newHTTPClient = origClient })

	var out bytes.Buffer
	err := captureStdout(&out, func() {
		require.NoError(t, doAPIRequest("http://example", "key", http.MethodPost, "/todos", map[string]any{"todos": []string{"a"}}))
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, received.method)
	assert.Equal(t, "/todos", received.path)
	assert.Equal(t, "key", received.header.Get("X-API-Key"))
	assert.Contains(t, received.body, `"todos"`)
	assert.Contains(t, out.String(), "201")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func captureStdout(dst *bytes.Buffer, fn func()) error {
	orig := stdOut
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	stdOut = w
	defer func() {
		stdOut = orig
	}()
	fn()
	_ = w.Close()
	_, _ = dst.ReadFrom(r)
	return nil
}
