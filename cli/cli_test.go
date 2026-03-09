package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/amharshit45/todos-cli-/todo"
)

type mockStorage struct {
	todos  []todo.Todo
	nextID int
}

func newMockStorage() *mockStorage {
	return &mockStorage{nextID: 1}
}

func (m *mockStorage) Add(_ context.Context, description string) error {
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	m.todos = append(m.todos, todo.Todo{ID: m.nextID, Description: description})
	m.nextID++
	return nil
}

func (m *mockStorage) List(_ context.Context) ([]todo.Todo, error) {
	result := make([]todo.Todo, len(m.todos))
	copy(result, m.todos)
	return result, nil
}

func (m *mockStorage) Delete(_ context.Context, id int) error {
	for i, t := range m.todos {
		if t.ID == id {
			m.todos = append(m.todos[:i], m.todos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) SetCompleted(_ context.Context, id int, completed bool) error {
	for i, t := range m.todos {
		if t.ID == id {
			m.todos[i].Completed = completed
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) Edit(_ context.Context, id int, description string) error {
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	for i, t := range m.todos {
		if t.ID == id {
			m.todos[i].Description = description
			return nil
		}
	}
	return fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (m *mockStorage) Close(_ context.Context) error {
	return nil
}

func runApp(t *testing.T, store todo.Storage, input string) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(input))
	app := New(store, scanner)
	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestExit(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "7\n")
}

func TestAddTodo(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "1\nbuy milk\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Description != "buy milk" {
		t.Fatalf("expected 'buy milk', got %q", store.todos[0].Description)
	}
}

func TestListTodos(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "1\ntask one\n1\ntask two\n2\n7\n")

	if len(store.todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(store.todos))
	}
}

func TestDeleteTodo(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "1\nto delete\n1\nto keep\n3\n1\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Description != "to keep" {
		t.Fatalf("expected 'to keep', got %q", store.todos[0].Description)
	}
}

func TestMarkCompleted(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "1\nmy task\n4\n1\n7\n")

	if !store.todos[0].Completed {
		t.Fatal("expected todo to be completed")
	}
}

func TestMarkIncomplete(t *testing.T) {
	store := newMockStorage()
	store.todos = append(store.todos, todo.Todo{ID: 1, Description: "done task", Completed: true})
	store.nextID = 2
	runApp(t, store, "5\n1\n7\n")

	if store.todos[0].Completed {
		t.Fatal("expected todo to be incomplete")
	}
}

func TestEditTodo(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "1\noriginal\n6\n1\nupdated\n7\n")

	if store.todos[0].Description != "updated" {
		t.Fatalf("expected 'updated', got %q", store.todos[0].Description)
	}
}

func TestInvalidMenuChoice(t *testing.T) {
	store := newMockStorage()
	runApp(t, store, "99\nabc\n7\n")
}

func TestContextCancellation(t *testing.T) {
	store := newMockStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scanner := bufio.NewScanner(strings.NewReader("7\n"))
	app := New(store, scanner)
	if err := app.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
