package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type ListResponse[T any] struct {
	Data    []T   `json:"data"`
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}

func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{Error: http.StatusText(status), Message: msg})
}

func ErrorWithRequest(w http.ResponseWriter, r *http.Request, status int, msg string) {
	reqID := chimw.GetReqID(r.Context())
	JSON(w, status, ErrorResponse{Error: http.StatusText(status), Message: msg, RequestID: reqID})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
