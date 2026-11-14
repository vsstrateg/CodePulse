package domain

import "errors"

var ErrTaskNotFound = errors.New("task not found")
var ErrCodeLimit = errors.New("code limit conditions not met")
var ErrUserIDLen = errors.New("user ID is too short")
var ErrInvalidTaskID = errors.New("invalid task id")
var ErrSubmissionNotFound = errors.New("submission not found")
