package todo

import (
	"fmt"
	"unicode/utf8"
)

const (
	MaxTitleLength       = 100
	MaxDescriptionLength = 500
)

type Todo struct {
	ID          int    `json:"id" bson:"_id"`
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	Completed   bool   `json:"completed" bson:"completed"`
}

func ValidateID(id int) error {
	if id <= 0 {
		return fmt.Errorf("id %d: %w", id, ErrInvalidID)
	}
	return nil
}

func ValidateTitle(title string) error {
	if title == "" {
		return ErrEmptyTitle
	}
	if n := utf8.RuneCountInString(title); n > MaxTitleLength {
		return fmt.Errorf("%w: %d characters (max %d)", ErrTitleTooLong, n, MaxTitleLength)
	}
	return nil
}

func ValidateDescription(desc string) error {
	if desc == "" {
		return nil
	}
	if n := utf8.RuneCountInString(desc); n > MaxDescriptionLength {
		return fmt.Errorf("%w: %d characters (max %d)", ErrDescriptionTooLong, n, MaxDescriptionLength)
	}
	return nil
}
