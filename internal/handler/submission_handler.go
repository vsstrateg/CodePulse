package handler

import (
	"codepulse/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"uuid"
)

type SubmissionService interface {
	CreateSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error)
	GetSubmissionByID(ctx context.Context, id string) (*domain.Submission, error)
}

type SubmissionHandler struct {
	service SubmissionService
}

func NewSubmissionHandler(service SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
}

func (h *SubmissionHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	submissionUUID, err := h.service.CreateSubmission(r.Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrCodeLimit) {
			sendError(w, http.StatusBadRequest, "code limit conditions not met")
			return
		}
		if errors.Is(err, domain.ErrUserIDLen) {
			sendError(w, http.StatusBadRequest, "user ID is too short")
			return
		}
		if errors.Is(err, domain.ErrInvalidTaskID) {
			sendError(w, http.StatusBadRequest, "invalid task id")
			return
		}
		if errors.Is(err, domain.ErrTaskNotFound) {
			sendError(w, http.StatusNotFound, "task not found")
			return
		}
		sendError(w, http.StatusInternalServerError, "internal server error")
		return

	}

	resp := domain.SubmissionResponse{
		SubmissionID: submissionUUID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
		return
	}

}

func (h *SubmissionHandler) GetSubmissionByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, err := uuid.Parse(id); err != nil {
		sendError(w, http.StatusBadRequest, "invalid submission id")
		return
	}

	sub, err := h.service.GetSubmissionByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSubmissionNotFound) {
			sendError(w, http.StatusNotFound, "submission not found")
			return
		}
		slog.Error("internal server error", "error", err)
		sendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sub); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
		return
	}

}
