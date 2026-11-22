package storage

import (
	"codepulse/internal/domain"
	"context"
	"database/sql"
	"errors"
	"log/slog"
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

func (s *SubmissionStorage) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	query := `UPDATE submissions SET status = $1, updated_at = NOW() WHERE id = $2`

	_, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *SubmissionStorage) UpdateResult(ctx context.Context, id string, res domain.ExecutionResult) error {
	query := `UPDATE submissions SET status = $1, execution_time_ms = $2, error_message = $3, updated_at = NOW() WHERE id = $4`

	_, err := s.db.ExecContext(ctx, query, res.Verdict, res.ExecutionTime, res.ErrorMessage, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *SubmissionStorage) FindSubmissionsByUserID(ctx context.Context, userID string) ([]domain.Submission, error) {
	query := `SELECT id, task_id, user_id, source_code, status, execution_time_ms, error_message, created_at, updated_at FROM submissions WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close submissions rows", "error", err)
		}
	}()

	var submissions []domain.Submission
	for rows.Next() {
		var submission domain.Submission
		if err := rows.Scan(&submission.ID, &submission.TaskID, &submission.UserID, &submission.SourceCode, &submission.Status, &submission.ExecutionTime, &submission.ErrorMessage, &submission.CreatedAt, &submission.UpdatedAt); err != nil {
			return nil, err
		}
		submissions = append(submissions, submission)

	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}
