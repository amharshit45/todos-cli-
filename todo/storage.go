package todo

import "context"

type Storage interface {
	Add(ctx context.Context, title, description string) error
	List(ctx context.Context) ([]Todo, error)
	Delete(ctx context.Context, id int) error
	SetCompleted(ctx context.Context, id int, completed bool) error
	EditTitle(ctx context.Context, id int, title string) error
	EditDescription(ctx context.Context, id int, description string) error
	Close(ctx context.Context) error
}
