package course

import "errors"

// Domain-specific errors for course operations.
var (
	ErrNotFound      = errors.New("course not found")
	ErrFetchFailed   = errors.New("failed to fetch course(s)")
	ErrProgressQuery = errors.New("failed to query user progress")
)
