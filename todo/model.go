package todo

import "fmt"

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
	if len(desc) > MaxDescriptionLength {
		return fmt.Errorf("%w: %d characters (max %d)", ErrDescriptionTooLong, len(desc), MaxDescriptionLength)
	}
	return nil
}
