package word

import "errors"

// Domain-specific errors for word operations.
var (
	ErrNotFound               = errors.New("word not found")
	ErrSourceNotFound         = errors.New("word source not found")
	ErrCourseWordLinkNotFound = errors.New("link between course and word not found for this index")
)
