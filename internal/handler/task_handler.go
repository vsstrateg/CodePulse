package handler

import (
	"codepulse/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type TaskService interface {
	CreateTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error)
	GetTasks(ctx context.Context) ([]domain.Task, error)
	GetTaskByID(ctx context.Context, id int) (*domain.TaskDetailResponse, error)
}

type TaskHandler struct {
	service TaskService
}

func NewTaskHandler(service TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	task, err := h.service.CreateTask(r.Context(), &req)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
		return
	}
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {

	tasks, err := h.service.GetTasks(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			sendError(w, http.StatusNotFound, "task not found")
			return
		}
		slog.Error("internal server error", "error", err)
		sendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
		return
	}
}
func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		sendError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.service.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			sendError(w, http.StatusNotFound, "task not found")
			return
		}
		slog.Error("internal server error", "error", err)
		sendError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		slog.Warn("the client terminated the connection", "error", err)
		return
	}

}
