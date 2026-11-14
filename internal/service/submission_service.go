package service

import (
	"codepulse/internal/domain"
	"context"
	"strings"
)

type SubmissionRepository interface {
	SaveSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error)
	FindSubmissionByID(ctx context.Context, id string) (*domain.Submission, error)
}

type SubmissionService struct {
	storage  SubmissionRepository
	taskRepo TaskRepository
}

func NewSubmissionService(storage SubmissionRepository, taskRepo TaskRepository) *SubmissionService {
	return &SubmissionService{
		storage:  storage,
		taskRepo: taskRepo,
	}
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error) {
	sourceCode := strings.TrimSpace(req.SourceCode)
	if len(sourceCode) < 1 || len(req.SourceCode) > 64*1024 {
		return "", domain.ErrCodeLimit
	}
	if len(req.UserID) < 1 {
		return "", domain.ErrUserIDLen
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

	return str, nil
}
func (s *SubmissionService) GetSubmissionByID(ctx context.Context, id string) (*domain.Submission, error) {
	sub, err := s.storage.FindSubmissionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
