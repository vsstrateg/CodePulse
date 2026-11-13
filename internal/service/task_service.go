package service

import (
	"codepulse/internal/domain"
	"context"
)

type TaskRepository interface {
	SaveTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error)
	FindTasks(ctx context.Context) ([]domain.Task, error)
	FindTaskByID(ctx context.Context, id int) (*domain.TaskDetailResponse, error)
}

type TaskService struct {
	storage TaskRepository
}

func NewTaskService(storage TaskRepository) *TaskService {
	return &TaskService{storage: storage}
}

func (t *TaskService) CreateTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	return t.storage.SaveTask(ctx, req)
}

func (t *TaskService) GetTasks(ctx context.Context) ([]domain.Task, error) {
	return t.storage.FindTasks(ctx)
}

func (t *TaskService) GetTaskByID(ctx context.Context, id int) (*domain.TaskDetailResponse, error) {
	return t.storage.FindTaskByID(ctx, id)
}
