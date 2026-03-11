package grpcclient

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/amharshit45/todos-cli-/gen/todopb"
	"github.com/amharshit45/todos-cli-/todo"
)

var _ todo.Storage = (*Storage)(nil)

type Storage struct {
	conn   *grpc.ClientConn
	client todopb.TodoServiceClient
}

func NewStorage(conn *grpc.ClientConn) *Storage {
	return &Storage{
		conn:   conn,
		client: todopb.NewTodoServiceClient(conn),
	}
}

func (s *Storage) Add(ctx context.Context, title, description string) error {
	_, err := s.client.Add(ctx, &todopb.AddRequest{Title: title, Description: description})
	return grpcToDomainError(err)
}

func (s *Storage) List(ctx context.Context) ([]todo.Todo, error) {
	resp, err := s.client.List(ctx, &todopb.ListRequest{})
	if err != nil {
		return nil, grpcToDomainError(err)
	}
	todos := make([]todo.Todo, len(resp.GetTodos()))
	for i, t := range resp.GetTodos() {
		todos[i] = todo.Todo{
			ID:          int(t.GetId()),
			Title:       t.GetTitle(),
			Description: t.GetDescription(),
			Completed:   t.GetCompleted(),
		}
	}
	return todos, nil
}

func (s *Storage) Delete(ctx context.Context, id int) error {
	_, err := s.client.Delete(ctx, &todopb.DeleteRequest{Id: int32(id)})
	return grpcToDomainError(err)
}

func (s *Storage) SetCompleted(ctx context.Context, id int, completed bool) error {
	_, err := s.client.SetCompleted(ctx, &todopb.SetCompletedRequest{
		Id:        int32(id),
		Completed: completed,
	})
	return grpcToDomainError(err)
}

func (s *Storage) EditTitle(ctx context.Context, id int, title string) error {
	_, err := s.client.EditTitle(ctx, &todopb.EditTitleRequest{
		Id:    int32(id),
		Title: title,
	})
	return grpcToDomainError(err)
}

func (s *Storage) EditDescription(ctx context.Context, id int, description string) error {
	_, err := s.client.EditDescription(ctx, &todopb.EditDescriptionRequest{
		Id:          int32(id),
		Description: description,
	})
	return grpcToDomainError(err)
}

func (s *Storage) Close(_ context.Context) error {
	return s.conn.Close()
}

// wrappedError preserves the original server message for display
// while wrapping the domain sentinel so errors.Is works across the gRPC boundary.
type wrappedError struct {
	msg      string
	sentinel error
}

func (e *wrappedError) Error() string { return e.msg }
func (e *wrappedError) Unwrap() error { return e.sentinel }

var codeToSentinels = map[codes.Code][]error{
	codes.NotFound: {todo.ErrNotFound},
	codes.FailedPrecondition: {
		todo.ErrAlreadyCompleted,
		todo.ErrAlreadyIncomplete,
		todo.ErrTitleUnchanged,
		todo.ErrDescriptionUnchanged,
	},
	codes.InvalidArgument: {
		todo.ErrInvalidID,
		todo.ErrEmptyTitle,
		todo.ErrTitleTooLong,
		todo.ErrDescriptionTooLong,
	},
}

func grpcToDomainError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	msg := st.Message()
	if sentinels, exists := codeToSentinels[st.Code()]; exists {
		for _, sentinel := range sentinels {
			if strings.Contains(msg, sentinel.Error()) {
				return &wrappedError{msg: msg, sentinel: sentinel}
			}
		}
	}

	return errors.New(msg)
}
