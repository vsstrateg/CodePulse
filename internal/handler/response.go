package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func sendError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: msg}); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
	}
}
