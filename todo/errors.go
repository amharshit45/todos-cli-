package todo

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyCompleted   = errors.New("already completed")
	ErrAlreadyIncomplete  = errors.New("already incomplete")
	ErrInvalidID          = errors.New("invalid ID")
	ErrEmptyDescription   = errors.New("description cannot be empty")
	ErrDescriptionTooLong = errors.New("description exceeds maximum length")
)
