package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"

	"github.com/amharshit45/todos-cli-/todo"
)

func TestMain(m *testing.M) {
	color.NoColor = true
	os.Exit(m.Run())
}

type mockStorage struct {
	todos  []todo.Todo
	nextID int
}

func newMockStorage() *mockStorage {
	return &mockStorage{nextID: 1}
}

func (m *mockStorage) Add(_ context.Context, description string) error {
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

func runApp(t *testing.T, store todo.Storage, input string) string {
	t.Helper()
	var buf bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader(input))
	app := New(store, scanner, &buf)
	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return buf.String()
}

func TestExit(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "7\n")

	if !strings.Contains(output, "===== Todo CLI =====") {
		t.Fatal("expected menu header in output")
	}
}

func TestAddTodo(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nbuy milk\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Description != "buy milk" {
		t.Fatalf("expected 'buy milk', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Todo added successfully.") {
		t.Fatalf("expected success message in output, got:\n%s", output)
	}
}

func TestListTodos(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\ntask one\n1\ntask two\n2\n7\n")

	if len(store.todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(store.todos))
	}
	if !strings.Contains(output, "task one") || !strings.Contains(output, "task two") {
		t.Fatalf("expected both tasks in list output, got:\n%s", output)
	}
}

func TestDeleteTodo(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nto delete\n1\nto keep\n3\n1\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Description != "to keep" {
		t.Fatalf("expected 'to keep', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Todo deleted successfully.") {
		t.Fatalf("expected delete message in output, got:\n%s", output)
	}
}

func TestMarkCompleted(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nmy task\n4\n1\n7\n")

	if !store.todos[0].Completed {
		t.Fatal("expected todo to be completed")
	}
	if !strings.Contains(output, "Todo marked as completed.") {
		t.Fatalf("expected completed message in output, got:\n%s", output)
	}
}

func TestMarkIncomplete(t *testing.T) {
	store := newMockStorage()
	store.todos = append(store.todos, todo.Todo{ID: 1, Description: "done task", Completed: true})
	store.nextID = 2
	output := runApp(t, store, "5\n1\n7\n")

	if store.todos[0].Completed {
		t.Fatal("expected todo to be incomplete")
	}
	if !strings.Contains(output, "Todo marked as incomplete.") {
		t.Fatalf("expected incomplete message in output, got:\n%s", output)
	}
}

func TestEditTodo(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\noriginal\n6\n1\nupdated\n7\n")

	if store.todos[0].Description != "updated" {
		t.Fatalf("expected 'updated', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Todo updated successfully.") {
		t.Fatalf("expected update message in output, got:\n%s", output)
	}
}

func TestInvalidMenuChoice(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "99\nabc\n7\n")

	if !strings.Contains(output, "Error: please enter a number between 1 and 7.") {
		t.Fatalf("expected invalid choice error in output, got:\n%s", output)
	}
}

func TestDeleteFromEmptyList(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "3\n7\n")

	if !strings.Contains(output, "No todos found.") {
		t.Fatalf("expected 'No todos found.' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Error: no todos to select from") {
		t.Fatalf("expected empty list error in output, got:\n%s", output)
	}
}

func TestContextCancellation(t *testing.T) {
	store := newMockStorage()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader("7\n"))
	app := New(store, scanner, &buf)
	if err := app.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
