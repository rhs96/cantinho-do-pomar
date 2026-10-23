package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]errorBody{"error": {Code: code, Message: message}})
}

func internalError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	logger.ErrorContext(r.Context(), "request failed",
		slog.String("error", err.Error()),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)
	writeError(w, http.StatusInternalServerError, "internal", "something went wrong")
}
