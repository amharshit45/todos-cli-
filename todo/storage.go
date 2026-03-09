package todo

import "context"

type Storage interface {
	Add(ctx context.Context, description string) error
	List(ctx context.Context) ([]Todo, error)
	Delete(ctx context.Context, id int) error
	SetCompleted(ctx context.Context, id int, completed bool) error
	Edit(ctx context.Context, id int, description string) error
	Close() error
}
