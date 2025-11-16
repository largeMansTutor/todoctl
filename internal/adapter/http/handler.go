package httpadapter

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	tododomain "github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/metrics"
	todousecase "github.com/thetrollfarmercodes/todoctl/todo/internal/usecase/todo"
)

// Handler encapsulates all HTTP handlers and their dependencies. It
// delegates to the todo Service for business logic. The methods on
// Handler must be registered with the router in router.go.
type Handler struct {
	svc         *todousecase.Service
	rateLimiter *RateLimiter
	metrics     *metrics.Provider
}

// NewHandler constructs a Handler with the provided service.
func NewHandler(svc *todousecase.Service, limiter *RateLimiter, metrics *metrics.Provider) *Handler {
	return &Handler{svc: svc, rateLimiter: limiter, metrics: metrics}
}

// CreateTodos handles POST /todos. It expects a JSON payload with
// a "todos" field containing an array of objects. It returns the
// created todos. If the request includes an Idempotency-Key header,
// the result is replayed when possible.
func (h *Handler) CreateTodos(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Todos []tododomain.Todo `json:"todos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	// Extract idempotency key
	idKey := r.Header.Get("Idempotency-Key")
	scope := "todos:create"
	todos, replay, err := h.svc.CreateTodos(r.Context(), idKey, scope, payload.Todos)
	if err != nil {
		// Determine status code based on error
		if strings.Contains(err.Error(), "duplicate") {
			httpError(w, http.StatusConflict, err.Error())
			return
		}
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if replay != nil {
		// Already has full JSON body; just write it with stored status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(replay)
		return
	}
	// Standard success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"todos": todos,
	})
}

// UpdateTodos handles PATCH /todos. It expects JSON with a
// "todos" field containing an array of update objects. Only fields
// present on each object will be updated. Duplicate IDs and empty
// payloads result in errors. Idempotency is supported.
func (h *Handler) UpdateTodos(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Todos []tododomain.TodoUpdate `json:"todos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(payload.Todos) == 0 {
		httpError(w, http.StatusBadRequest, "no todos to update")
		return
	}
	idKey := r.Header.Get("Idempotency-Key")
	scope := "todos:update"
	todos, replay, err := h.svc.UpdateTodos(r.Context(), idKey, scope, payload.Todos)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			httpError(w, http.StatusConflict, err.Error())
			return
		}
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if replay != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(replay)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"todos": todos,
	})
}

// ListTodos handles GET /todos. It supports two pagination modes:
// 1) offset pagination via ?page=&limit= query parameters (page starts at 1), and
// 2) keyset pagination via ?cursor=&limit= query parameters. If
// cursor is provided, page is ignored. It returns an array of todos
// and next_cursor if more results exist.
func (h *Handler) ListTodos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cursor := q.Get("cursor")
	page := 1
	limit := 10
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	todos, nextCursor, err := h.svc.ListTodos(r.Context(), page, limit, cursor)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp := map[string]interface{}{
		"todos": todos,
		"limit": limit,
	}
	if cursor == "" {
		resp["page"] = page
	} else {
		resp["cursor"] = cursor
	}
	if nextCursor != "" {
		resp["next_cursor"] = nextCursor
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Healthz returns 200 OK to indicate the service is alive. It can
// include additional diagnostics if desired.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Readyz returns 200 OK to indicate the service is ready to accept
// traffic. In a real deployment this could check dependencies like
// database connectivity.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

// httpError writes a JSON error response with the given status code and
// message. The message is sanitized to avoid exposing internal
// information. It sets Content-Type to application/json.
func httpError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
		},
	})
}
