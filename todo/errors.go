package todo

import "errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrAlreadyCompleted     = errors.New("already completed")
	ErrAlreadyIncomplete    = errors.New("already incomplete")
	ErrTitleUnchanged       = errors.New("title unchanged")
	ErrDescriptionUnchanged = errors.New("description unchanged")
	ErrInvalidID            = errors.New("invalid ID")
	ErrEmptyTitle           = errors.New("title cannot be empty")
	ErrTitleTooLong         = errors.New("title exceeds maximum length")
	ErrDescriptionTooLong   = errors.New("description exceeds maximum length")
)
