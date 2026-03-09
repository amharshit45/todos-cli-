package todo

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyCompleted  = errors.New("already completed")
	ErrAlreadyIncomplete = errors.New("already incomplete")
	ErrInvalidID         = errors.New("invalid ID")
)
