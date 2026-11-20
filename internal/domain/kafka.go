package domain

type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	TaskID       int    `json:"task_id"`
	UserID       string `json:"user_id"`
}
