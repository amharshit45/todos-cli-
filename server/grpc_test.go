package server_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/amharshit45/todos-cli-/gen/todopb"
	"github.com/amharshit45/todos-cli-/grpcclient"
	"github.com/amharshit45/todos-cli-/server"
	"github.com/amharshit45/todos-cli-/todo"
)

type mockStorage struct {
	todos  []todo.Todo
	nextID int
}

func newMockStorage() *mockStorage {
	return &mockStorage{nextID: 1}
}

func (m *mockStorage) Add(_ context.Context, title, description string) error {
	if err := todo.ValidateTitle(title); err != nil {
		return err
	}
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	m.todos = append(m.todos, todo.Todo{ID: m.nextID, Title: title, Description: description})
	m.nextID++
	return nil
}

func (m *mockStorage) List(_ context.Context) ([]todo.Todo, error) {
	result := make([]todo.Todo, len(m.todos))
	copy(result, m.todos)
	return result, nil
}

func (m *mockStorage) Delete(_ context.Context, id int) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	for i, t := range m.todos {
		if t.ID == id {
			m.todos = append(m.todos[:i], m.todos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) SetCompleted(_ context.Context, id int, completed bool) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	for i, t := range m.todos {
		if t.ID == id {
			if t.Completed == completed {
				if completed {
					return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyCompleted)
				}
				return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyIncomplete)
			}
			m.todos[i].Completed = completed
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) EditTitle(_ context.Context, id int, title string) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	if err := todo.ValidateTitle(title); err != nil {
		return err
	}
	for i, t := range m.todos {
		if t.ID == id {
			if t.Title == title {
				return fmt.Errorf("todo %d: %w", id, todo.ErrTitleUnchanged)
			}
			m.todos[i].Title = title
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) EditDescription(_ context.Context, id int, description string) error {
	if err := todo.ValidateID(id); err != nil {
		return err
	}
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	for i, t := range m.todos {
		if t.ID == id {
			if t.Description == description {
				return fmt.Errorf("todo %d: %w", id, todo.ErrDescriptionUnchanged)
			}
			m.todos[i].Description = description
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) Close(_ context.Context) error {
	return nil
}

const bufSize = 1024 * 1024

type testEnv struct {
	store  *mockStorage
	client todopb.TodoServiceClient
	conn   *grpc.ClientConn
	srv    *grpc.Server
}

func setup(t *testing.T) *testEnv {
	t.Helper()

	store := newMockStorage()
	lis := bufconn.Listen(bufSize)

	srv := grpc.NewServer()
	todopb.RegisterTodoServiceServer(srv, server.New(store))

	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Logf("Server exited: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufconn: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
		srv.GracefulStop()
	})

	return &testEnv{
		store:  store,
		client: todopb.NewTodoServiceClient(conn),
		conn:   conn,
		srv:    srv,
	}
}

func TestAdd(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.Add(ctx, &todopb.AddRequest{Title: "buy milk", Description: "from store"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if len(env.store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(env.store.todos))
	}
	if env.store.todos[0].Title != "buy milk" {
		t.Fatalf("expected title 'buy milk', got %q", env.store.todos[0].Title)
	}
	if env.store.todos[0].Description != "from store" {
		t.Fatalf("expected description 'from store', got %q", env.store.todos[0].Description)
	}
}

func TestAddNoDescription(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.Add(ctx, &todopb.AddRequest{Title: "buy milk"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if env.store.todos[0].Description != "" {
		t.Fatalf("expected empty description, got %q", env.store.todos[0].Description)
	}
}

func TestAddEmptyTitle(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.Add(ctx, &todopb.AddRequest{Title: "", Description: "some desc"})
	if err == nil {
		t.Fatal("expected error for empty title")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestList(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{
		{ID: 1, Title: "task one", Description: "details"},
		{ID: 2, Title: "task two", Completed: true},
	}

	resp, err := env.client.List(ctx, &todopb.ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(resp.GetTodos()) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(resp.GetTodos()))
	}

	first := resp.GetTodos()[0]
	if first.GetId() != 1 || first.GetTitle() != "task one" || first.GetDescription() != "details" || first.GetCompleted() {
		t.Fatalf("unexpected first todo: %+v", first)
	}

	second := resp.GetTodos()[1]
	if second.GetId() != 2 || second.GetTitle() != "task two" || !second.GetCompleted() {
		t.Fatalf("unexpected second todo: %+v", second)
	}
}

func TestListEmpty(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	resp, err := env.client.List(ctx, &todopb.ListRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(resp.GetTodos()) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(resp.GetTodos()))
	}
}

func TestDelete(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "to delete"}}

	_, err := env.client.Delete(ctx, &todopb.DeleteRequest{Id: 1})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(env.store.todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(env.store.todos))
	}
}

func TestDeleteNotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.Delete(ctx, &todopb.DeleteRequest{Id: 999})
	if err == nil {
		t.Fatal("expected error for deleting nonexistent todo")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestDeleteInvalidID(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.Delete(ctx, &todopb.DeleteRequest{Id: 0})
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestSetCompleted(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "my task"}}

	_, err := env.client.SetCompleted(ctx, &todopb.SetCompletedRequest{Id: 1, Completed: true})
	if err != nil {
		t.Fatalf("SetCompleted: %v", err)
	}
	if !env.store.todos[0].Completed {
		t.Fatal("expected todo to be completed")
	}
}

func TestSetCompletedAlready(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "done", Completed: true}}

	_, err := env.client.SetCompleted(ctx, &todopb.SetCompletedRequest{Id: 1, Completed: true})
	if err == nil {
		t.Fatal("expected error for already completed")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestSetCompletedNotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.SetCompleted(ctx, &todopb.SetCompletedRequest{Id: 999, Completed: true})
	if err == nil {
		t.Fatal("expected error for nonexistent todo")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestEditTitle(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "original"}}

	_, err := env.client.EditTitle(ctx, &todopb.EditTitleRequest{Id: 1, Title: "updated"})
	if err != nil {
		t.Fatalf("EditTitle: %v", err)
	}
	if env.store.todos[0].Title != "updated" {
		t.Fatalf("expected 'updated', got %q", env.store.todos[0].Title)
	}
}

func TestEditTitleNotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.EditTitle(ctx, &todopb.EditTitleRequest{Id: 999, Title: "updated"})
	if err == nil {
		t.Fatal("expected error for nonexistent todo")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestEditTitleEmpty(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "original"}}

	_, err := env.client.EditTitle(ctx, &todopb.EditTitleRequest{Id: 1, Title: ""})
	if err == nil {
		t.Fatal("expected error for empty title")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestEditTitleUnchanged(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "same"}}

	_, err := env.client.EditTitle(ctx, &todopb.EditTitleRequest{Id: 1, Title: "same"})
	if err == nil {
		t.Fatal("expected error for unchanged title")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestEditDescription(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "task", Description: "old"}}

	_, err := env.client.EditDescription(ctx, &todopb.EditDescriptionRequest{Id: 1, Description: "new"})
	if err != nil {
		t.Fatalf("EditDescription: %v", err)
	}
	if env.store.todos[0].Description != "new" {
		t.Fatalf("expected 'new', got %q", env.store.todos[0].Description)
	}
}

func TestEditDescriptionNotFound(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.client.EditDescription(ctx, &todopb.EditDescriptionRequest{Id: 999, Description: "new"})
	if err == nil {
		t.Fatal("expected error for nonexistent todo")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", st.Code())
	}
}

func TestEditDescriptionUnchanged(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.store.todos = []todo.Todo{{ID: 1, Title: "task", Description: "same"}}

	_, err := env.client.EditDescription(ctx, &todopb.EditDescriptionRequest{Id: 1, Description: "same"})
	if err == nil {
		t.Fatal("expected error for unchanged description")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
}

// TestRoundTripErrorMapping verifies domain errors survive the
// server->gRPC->client round-trip with errors.Is semantics intact.
func TestRoundTripErrorMapping(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	store := grpcclient.NewStorage(env.conn)

	env.store.todos = []todo.Todo{{ID: 1, Title: "task", Description: "desc", Completed: true}}

	tests := []struct {
		name     string
		fn       func() error
		sentinel error
	}{
		{
			name:     "delete not found",
			fn:       func() error { return store.Delete(ctx, 999) },
			sentinel: todo.ErrNotFound,
		},
		{
			name:     "already completed",
			fn:       func() error { return store.SetCompleted(ctx, 1, true) },
			sentinel: todo.ErrAlreadyCompleted,
		},
		{
			name:     "invalid ID",
			fn:       func() error { return store.Delete(ctx, 0) },
			sentinel: todo.ErrInvalidID,
		},
		{
			name:     "empty title",
			fn:       func() error { return store.Add(ctx, "", "desc") },
			sentinel: todo.ErrEmptyTitle,
		},
		{
			name:     "title unchanged",
			fn:       func() error { return store.EditTitle(ctx, 1, "task") },
			sentinel: todo.ErrTitleUnchanged,
		},
		{
			name:     "description unchanged",
			fn:       func() error { return store.EditDescription(ctx, 1, "desc") },
			sentinel: todo.ErrDescriptionUnchanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.sentinel) {
				t.Fatalf("expected errors.Is(%v, %v) to be true, got error: %v", err, tt.sentinel, err)
			}
		})
	}
}
