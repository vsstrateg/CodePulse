package domain

import "time"

type Submission struct {
	ID            string           `json:"id"`
	TaskID        int              `json:"task_id"`
	UserID        string           `json:"user_id"`
	SourceCode    string           `json:"source_code"`
	Status        SubmissionStatus `json:"status"`
	ExecutionTime *int             `json:"execution_time_ms,omitempty"`
	ErrorMessage  *string          `json:"error_message,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type SubmissionResponse struct {
	SubmissionID string `json:"submission_id"`
}

type SubmissionStatus string

const (
	StatusPending             SubmissionStatus = "PENDING"
	StatusRunning             SubmissionStatus = "RUNNING"
	StatusAccepted            SubmissionStatus = "ACCEPTED"
	StatusWrongAnswer         SubmissionStatus = "WRONG_ANSWER"
	StatusTimeLimitExceeded   SubmissionStatus = "TIME_LIMIT_EXCEEDED"
	StatusMemoryLimitExceeded SubmissionStatus = "MEMORY_LIMIT_EXCEEDED"
	StatusCompilationError    SubmissionStatus = "COMPILATION_ERROR"
	StatusRuntimeError        SubmissionStatus = "RUNTIME_ERROR"
)
