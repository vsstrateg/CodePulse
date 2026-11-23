package service

import (
	"codepulse/internal/domain"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type mockSubRepo struct {
	saveFn func(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error)
}
type mockTaskRepo struct {
	findFn func(ctx context.Context, id int) (*domain.TaskDetailResponse, error)
}
type mockQueue struct {
	published bool
}
type mockLimiter struct {
	allow bool
}

func (m *mockSubRepo) SaveSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error) {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return "sub-123", nil
}
func (m *mockSubRepo) FindSubmissionByID(ctx context.Context, id string) (*domain.Submission, error) {
	return nil, nil
}
func (m *mockSubRepo) FindSubmissionsByUserID(ctx context.Context, userID string) ([]domain.Submission, error) {
	return nil, nil
}

func (m *mockTaskRepo) SaveTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) FindTasks(ctx context.Context) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) FindTaskByID(ctx context.Context, id int) (*domain.TaskDetailResponse, error) {
	if m.findFn != nil {
		return m.findFn(ctx, id)
	}
	return &domain.TaskDetailResponse{}, nil
}

func (m *mockQueue) Publish(ctx context.Context, event domain.SubmissionEvent) error {
	m.published = true
	return nil
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	return m.allow, nil
}

func TestCreateSubmission(t *testing.T) {
	validUserID := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

	t.Run("Empty code returns ErrCodeLimit", func(t *testing.T) {
		svc := NewSubmissionService(&mockSubRepo{}, &mockTaskRepo{}, &mockQueue{}, &mockLimiter{allow: true})
		_, err := svc.CreateSubmission(context.Background(), &domain.CreateSubmissionRequest{
			TaskID:     1,
			UserID:     validUserID,
			SourceCode: "   ",
		})
		if !errors.Is(err, domain.ErrCodeLimit) {
			t.Fatalf("expected ErrCodeLimit, got %v", err)
		}
	})

	t.Run("Code too large returns ErrCodeLimit", func(t *testing.T) {
		svc := NewSubmissionService(&mockSubRepo{}, &mockTaskRepo{}, &mockQueue{}, &mockLimiter{allow: true})
		hugeCode := strings.Repeat("a", 65*1024)
		_, err := svc.CreateSubmission(context.Background(), &domain.CreateSubmissionRequest{
			TaskID:     1,
			UserID:     validUserID,
			SourceCode: hugeCode,
		})
		if !errors.Is(err, domain.ErrCodeLimit) {
			t.Fatalf("expected ErrCodeLimit, got %v", err)
		}
	})

	t.Run("Invalid UserID returns ErrInvalidUserID", func(t *testing.T) {
		svc := NewSubmissionService(&mockSubRepo{}, &mockTaskRepo{}, &mockQueue{}, &mockLimiter{allow: true})
		_, err := svc.CreateSubmission(context.Background(), &domain.CreateSubmissionRequest{
			TaskID:     1,
			UserID:     "not-a-uuid",
			SourceCode: "package main",
		})
		if !errors.Is(err, domain.ErrInvalidUserID) {
			t.Fatalf("expected ErrInvalidUserID, got %v", err)
		}
	})

	t.Run("Rate limit exceeded returns ErrRateLimitExceeded", func(t *testing.T) {
		svc := NewSubmissionService(&mockSubRepo{}, &mockTaskRepo{}, &mockQueue{}, &mockLimiter{allow: false})
		_, err := svc.CreateSubmission(context.Background(), &domain.CreateSubmissionRequest{
			TaskID:     1,
			UserID:     validUserID,
			SourceCode: "package main",
		})
		if !errors.Is(err, domain.ErrRateLimitExceeded) {
			t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
		}
	})

	t.Run("Successful submission saves and publishes to queue", func(t *testing.T) {
		queue := &mockQueue{}
		svc := NewSubmissionService(&mockSubRepo{}, &mockTaskRepo{}, queue, &mockLimiter{allow: true})
		id, err := svc.CreateSubmission(context.Background(), &domain.CreateSubmissionRequest{
			TaskID:     1,
			UserID:     validUserID,
			SourceCode: "package main\nfunc main() {}",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "sub-123" {
			t.Fatalf("expected id sub-123, got %s", id)
		}
		if !queue.published {
			t.Fatal("expected message to be published to queue")
		}
	})
}
