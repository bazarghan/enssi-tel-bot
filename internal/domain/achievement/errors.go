package achievement

import "errors"

// Domain-specific errors for achievement operations.
var (
	ErrNotFound        = errors.New("achievement not found")
	ErrUserAchNotFound = errors.New("user achievement progress not found")
	ErrSaveFailed      = errors.New("failed to save user achievement progress")
)
