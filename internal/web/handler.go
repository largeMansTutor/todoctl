package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/core/todo"
	"github.com/thetrollfarmercodes/todoctl/todo/pkg/httpadapter"
	"go.uber.org/zap"
)

// CreateTodos handles POST /todos. It expects a JSON payload with
// a "todos" field containing an array of objects. It returns the
// created todos. If the request includes an Idempotency-Key header,
// the result is replayed when possible.
func (h *App) CreateTodos(w http.ResponseWriter, r *http.Request) {
	log := h.loggerFor(r)
	var payload struct {
		Todos DedupTodos `json:"todos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Warn("create todos: invalid json", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	// Extract idempotency key
	idKey := r.Header.Get("Idempotency-Key")
	scope := "todos:create"
	todos, replay, err := h.svc.CreateTodos(r.Context(), idKey, scope, payload.Todos)
	if err != nil {
		// Determine status code based on error
		if errors.Is(err, todo.ErrDuplicateTitle) {
			log.Warn("create todos conflict", zap.Error(err))
			httpadapter.HttpError(w, http.StatusConflict, "conflict")
			return
		}
		log.Error("create todos failed", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid request")
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

// GetTodo handles GET /todo? id or title. Returns 400 for missing params and
// 404 when no todo is found.
func (h *App) GetTodo(w http.ResponseWriter, r *http.Request) {
	log := h.loggerFor(r)
	q := r.URL.Query()
	idStr := chi.URLParam(r, "id")
	title := q.Get("title")
	var id uint64
	if idStr != "" {
		if parsed, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			id = parsed
		} else {
			log.Warn("get todo: invalid id", zap.Error(err))
			httpadapter.HttpError(w, http.StatusBadRequest, "invalid request")
			return
		}
	}
	todoItem, err := h.svc.GetTodo(r.Context(), id, title)
	if err != nil {
		if errors.Is(err, todo.ErrNotFound) {
			log.Warn("get todo: not found", zap.Error(err))
			httpadapter.HttpError(w, http.StatusNotFound, "not found")
			return
		}
		log.Error("get todo failed", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid request")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"todo": todoItem,
	})
}

// UpdateTodos handles PATCH /todos. It expects JSON with a
// "todos" field containing an array of update objects. Only fields
// present on each object will be updated. Duplicate IDs and empty
// payloads result in errors. Idempotency is supported.
func (h *App) UpdateTodos(w http.ResponseWriter, r *http.Request) {
	log := h.loggerFor(r)
	var payload struct {
		Todos DedupTodosUpdate `json:"todos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Warn("update todos: invalid json", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(payload.Todos) == 0 {
		log.Warn("update todos: empty payload")
		httpadapter.HttpError(w, http.StatusBadRequest, "no todos to update")
		return
	}
	idKey := r.Header.Get("Idempotency-Key")
	scope := "todos:update"
	todos, replay, err := h.svc.UpdateTodos(r.Context(), idKey, scope, payload.Todos)
	if err != nil {
		if errors.Is(err, todo.ErrDuplicateTitle) {
			log.Warn("update todos conflict", zap.Error(err))
			httpadapter.HttpError(w, http.StatusConflict, "conflict")
			return
		}
		log.Error("update todos failed", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid request")
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
func (h *App) ListTodos(w http.ResponseWriter, r *http.Request) {
	log := h.loggerFor(r)
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
		log.Warn("list todos failed", zap.Error(err))
		httpadapter.HttpError(w, http.StatusBadRequest, "invalid request")
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

func (h *App) loggerFor(r *http.Request) *zap.Logger {
	log := h.logger
	if log == nil {
		return zap.NewNop()
	}
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		log = log.With(zap.String("request_id", reqID))
	}
	return log.With(
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)
}
