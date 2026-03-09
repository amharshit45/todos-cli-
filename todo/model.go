package todo

import (
	"fmt"
	"unicode/utf8"
)

const MaxDescriptionLength = 500

type Todo struct {
	ID          int    `json:"id" bson:"_id"`
	Description string `json:"description" bson:"description"`
	Completed   bool   `json:"completed" bson:"completed"`
}

func ValidateID(id int) error {
	if id <= 0 {
		return fmt.Errorf("id %d: %w", id, ErrInvalidID)
	}
	return nil
}

func ValidateDescription(desc string) error {
	if desc == "" {
		return ErrEmptyDescription
	}
	if n := utf8.RuneCountInString(desc); n > MaxDescriptionLength {
		return fmt.Errorf("%w: %d characters (max %d)", ErrDescriptionTooLong, n, MaxDescriptionLength)
	}
	return nil
}
