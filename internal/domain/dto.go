package domain

type CreateTestRequest struct {
	Input          string `json:"input_data"`
	ExpectedOutput string `json:"expected_output"`
	IsSample       bool   `json:"is_sample"`
}

type CreateTaskRequest struct {
	Title         string              `json:"title"`
	Description   string              `json:"description"`
	TimeLimitMs   int                 `json:"time_limit_ms"`
	MemoryLimitMb int                 `json:"memory_limit_mb"`
	Points        int                 `json:"points"`
	Tests         []CreateTestRequest `json:"tests"`
}

type CreateSubmissionRequest struct {
	TaskID     int    `json:"task_id"`
	UserID     string `json:"user_id"`
	SourceCode string `json:"source_code"`
}

type TaskDetailResponse struct {
	Task
	Tests []Test `json:"tests"`
}

type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	TaskID       int    `json:"task_id"`
	UserID       string `json:"user_id"`
}
