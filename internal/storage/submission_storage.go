package storage

import (
	"codepulse/internal/domain"
	"context"
	"database/sql"
	"errors"
)

type SubmissionStorage struct {
	db *sql.DB
}

func NewSubmissionStorage(db *sql.DB) *SubmissionStorage {
	return &SubmissionStorage{db: db}
}

func (s *SubmissionStorage) SaveSubmission(ctx context.Context, req *domain.CreateSubmissionRequest) (string, error) {

	var submissionUUID string

	query := `INSERT INTO submissions (task_id, user_id, source_code, status) VALUES ($1, $2, $3, $4) RETURNING id`

	err := s.db.QueryRowContext(ctx, query, req.TaskID, req.UserID, req.SourceCode, domain.StatusPending).Scan(&submissionUUID)
	if err != nil {
		return "", err
	}

	return submissionUUID, nil
}

func (s *SubmissionStorage) FindSubmissionByID(ctx context.Context, id string) (*domain.Submission, error) {
	query := `SELECT id, task_id, user_id, source_code, status, execution_time_ms, error_message, created_at, updated_at FROM submissions WHERE id = $1`

	row := s.db.QueryRowContext(ctx, query, id)

	var submission domain.Submission

	if err := row.Scan(&submission.ID, &submission.TaskID, &submission.UserID, &submission.SourceCode, &submission.Status, &submission.ExecutionTime, &submission.ErrorMessage, &submission.CreatedAt, &submission.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, err
	}

	return &submission, nil
}
