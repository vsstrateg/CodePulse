package storage

import (
	"codepulse/internal/domain"
	"context"
	"database/sql"
	"errors"
	"log/slog"
)

var ErrNotFound = errors.New("task not found")

type TaskStorage struct {
	db *sql.DB
}

func NewTaskStorage(db *sql.DB) *TaskStorage {
	return &TaskStorage{db: db}
}

func (s *TaskStorage) SaveTask(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			slog.Error("failed to Rollback tx", "error", err)
		}
	}()

	var taskID int

	query := `INSERT INTO tasks (title, description, time_limit_ms, memory_limit_mb, points) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err = tx.QueryRowContext(ctx, query, req.Title, req.Description, req.TimeLimitMs, req.MemoryLimitMb, req.Points).Scan(&taskID)
	if err != nil {
		return nil, err
	}

	for _, test := range req.Tests {
		testQuery := `INSERT INTO task_tests (task_id, input_data, expected_output, is_sample) VALUES ($1, $2, $3, $4)`
		_, err := tx.ExecContext(ctx, testQuery, taskID, test.Input, test.ExpectedOutput, test.IsSample)
		if err != nil {
			return nil, err
		}

	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &domain.Task{
		ID:            taskID,
		Title:         req.Title,
		Description:   req.Description,
		TimeLimitMs:   req.TimeLimitMs,
		MemoryLimitMb: req.MemoryLimitMb,
		Points:        req.Points,
	}, nil
}
func (s *TaskStorage) FindTasks(ctx context.Context) ([]domain.Task, error) {
	query := `SELECT id, title, description, time_limit_ms, memory_limit_mb, points FROM tasks ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.TimeLimitMs, &task.MemoryLimitMb, &task.Points); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *TaskStorage) FindTaskByID(ctx context.Context, id int) (*domain.TaskDetailResponse, error) {
	query := `SELECT id, title, description, time_limit_ms, memory_limit_mb, points FROM tasks WHERE id = $1`

	row := s.db.QueryRowContext(ctx, query, id)

	var task domain.Task
	if err := row.Scan(&task.ID, &task.Title, &task.Description, &task.TimeLimitMs, &task.MemoryLimitMb, &task.Points); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := row.Err(); err != nil {
		return nil, err
	}

	queryTests := `SELECT id, task_id, input_data, expected_output, is_sample FROM task_tests WHERE task_id = $1 AND is_sample = true`

	tests, err := s.db.QueryContext(ctx, queryTests, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tests.Close(); err != nil {
			slog.Error("failed to close tests rows", "error", err)
		}
	}()

	var testsArr []domain.Test
	for tests.Next() {
		var test domain.Test
		if err := tests.Scan(&test.ID, &test.TaskID, &test.Input, &test.ExpectedOutput, &test.IsSample); err != nil {
			return nil, err
		}
		testsArr = append(testsArr, test)
	}
	if err := tests.Err(); err != nil {
		return nil, err
	}

	return &domain.TaskDetailResponse{
		Task:  task,
		Tests: testsArr,
	}, nil
}
