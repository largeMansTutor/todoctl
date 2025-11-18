package httpadapter

import (
	"encoding/json"
	"net/http"
)

// Health returns 200 OK to indicate the service is alive. It can
// include additional diagnostics if desired.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// HttpError writes a JSON error response with the given status code and
// message. The message is sanitized to avoid exposing internal
// information. It sets Content-Type to application/json.
func HttpError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": ErrorResponse{
			"message": msg,
		},
	})
}

type ErrorResponse = map[string]any
