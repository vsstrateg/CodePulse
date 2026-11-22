package service

import (
	"codepulse/internal/domain"
	"context"
	"strings"
	"uuid"
)

type SubmissionRepository interface {
	SaveSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error)
	FindSubmissionByID(ctx context.Context, id string) (*domain.Submission, error)
	FindSubmissionsByUserID(ctx context.Context, userID string) ([]domain.Submission, error)
}

type SubmissionQueue interface {
	Publish(ctx context.Context, event domain.SubmissionEvent) error
}

type SubmissionService struct {
	storage  SubmissionRepository
	taskRepo TaskRepository
	queue    SubmissionQueue
}

func NewSubmissionService(storage SubmissionRepository, taskRepo TaskRepository, queue SubmissionQueue) *SubmissionService {
	return &SubmissionService{
		storage:  storage,
		taskRepo: taskRepo,
		queue:    queue,
	}
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error) {
	sourceCode := strings.TrimSpace(req.SourceCode)
	if len(sourceCode) < 1 || len(req.SourceCode) > 64*1024 {
		return "", domain.ErrCodeLimit
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		return "", domain.ErrInvalidUserID
	}
	if req.TaskID < 1 {
		return "", domain.ErrInvalidTaskID
	}

	if _, err := s.taskRepo.FindTaskByID(ctx, req.TaskID); err != nil {
		return "", err
	}

	str, err := s.storage.SaveSubmission(ctx, req)
	if err != nil {
		return "", err
	}

	event := domain.SubmissionEvent{
		SubmissionID: str,
		TaskID:       req.TaskID,
		UserID:       req.UserID,
	}

	if err = s.queue.Publish(ctx, event); err != nil {
		return "", err
	}

	return str, nil
}
func (s *SubmissionService) GetSubmissionByID(ctx context.Context, id string) (*domain.Submission, error) {
	sub, err := s.storage.FindSubmissionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *SubmissionService) GetSubmissionsByUserID(ctx context.Context, userID string) ([]domain.Submission, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, domain.ErrInvalidUserID
	}

	submissions, err := s.storage.FindSubmissionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return submissions, nil
}
