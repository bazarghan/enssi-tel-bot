package user

import (
	"errors"
)

var (
	ErrUserNotFound = errors.New("domain::user: user not found")
)
