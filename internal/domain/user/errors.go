package user

import "errors"

// Domain-specific errors for user operations.
var (
	ErrNotFound      = errors.New("user not found")
	ErrProfileNotSet = errors.New("user profile is not set")
	ErrSaveFailed    = errors.New("failed to save user")
	ErrCreateFailed  = errors.New("failed to create new user and profile")
	ErrInvalidInput  = errors.New("invalid input provided for user operation")
)
