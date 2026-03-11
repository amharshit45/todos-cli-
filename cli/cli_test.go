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

func (m *mockStorage) Add(_ context.Context, title, description string) error {
	if err := todo.ValidateTitle(title); err != nil {
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

func TestHelpMenu(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "0\n7\n")

	count := strings.Count(output, "===== Todo CLI =====")
	if count != 2 {
		t.Fatalf("expected menu printed twice, got %d times", count)
	}
}

func TestAddTodo(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nbuy milk\nfrom the store\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Title != "buy milk" {
		t.Fatalf("expected title 'buy milk', got %q", store.todos[0].Title)
	}
	if store.todos[0].Description != "from the store" {
		t.Fatalf("expected description 'from the store', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Todo added successfully.") {
		t.Fatalf("expected success message in output, got:\n%s", output)
	}
}

func TestAddTodoNoDescription(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nbuy milk\n\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Title != "buy milk" {
		t.Fatalf("expected title 'buy milk', got %q", store.todos[0].Title)
	}
	if store.todos[0].Description != "" {
		t.Fatalf("expected empty description, got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Todo added successfully.") {
		t.Fatalf("expected success message in output, got:\n%s", output)
	}
}

func TestListTodos(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\ntask one\ndetails\n1\ntask two\n\n2\n7\n")

	if len(store.todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(store.todos))
	}
	if !strings.Contains(output, "task one - details") {
		t.Fatalf("expected 'task one - details' in list output, got:\n%s", output)
	}
	if !strings.Contains(output, "task two") {
		t.Fatalf("expected 'task two' in list output, got:\n%s", output)
	}
}

func TestDeleteTodo(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nto delete\n\n1\nto keep\n\n3\n1\n7\n")

	if len(store.todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(store.todos))
	}
	if store.todos[0].Title != "to keep" {
		t.Fatalf("expected 'to keep', got %q", store.todos[0].Title)
	}
	if !strings.Contains(output, "Todo deleted successfully.") {
		t.Fatalf("expected delete message in output, got:\n%s", output)
	}
}

func TestMarkCompleted(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nmy task\n\n4\n1\n7\n")

	if !store.todos[0].Completed {
		t.Fatal("expected todo to be completed")
	}
	if !strings.Contains(output, "Todo marked as completed.") {
		t.Fatalf("expected completed message in output, got:\n%s", output)
	}
}

func TestMarkIncomplete(t *testing.T) {
	store := newMockStorage()
	store.todos = append(store.todos, todo.Todo{ID: 1, Title: "done task", Completed: true})
	store.nextID = 2
	output := runApp(t, store, "5\n1\n7\n")

	if store.todos[0].Completed {
		t.Fatal("expected todo to be incomplete")
	}
	if !strings.Contains(output, "Todo marked as incomplete.") {
		t.Fatalf("expected incomplete message in output, got:\n%s", output)
	}
}

func TestAlreadyCompleted(t *testing.T) {
	store := newMockStorage()
	store.todos = append(store.todos, todo.Todo{ID: 1, Title: "done", Completed: true})
	store.nextID = 2
	output := runApp(t, store, "4\n1\n7\n")

	if !strings.Contains(output, "Info: todo 1 is already completed.") {
		t.Fatalf("expected info message, got:\n%s", output)
	}
}

func TestEditTitle(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\noriginal\n\n6\n1\nt\nupdated\n7\n")

	if store.todos[0].Title != "updated" {
		t.Fatalf("expected 'updated', got %q", store.todos[0].Title)
	}
	if !strings.Contains(output, "Title updated successfully.") {
		t.Fatalf("expected title update message in output, got:\n%s", output)
	}
}

func TestEditBoth(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nold title\nold desc\n6\n1\nb\nnew title\nnew desc\n7\n")

	if store.todos[0].Title != "new title" {
		t.Fatalf("expected title 'new title', got %q", store.todos[0].Title)
	}
	if store.todos[0].Description != "new desc" {
		t.Fatalf("expected description 'new desc', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Title updated successfully.") {
		t.Fatalf("expected title update message in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Description updated successfully.") {
		t.Fatalf("expected description update message in output, got:\n%s", output)
	}
}

func TestEditDescription(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nmy task\nold desc\n6\n1\nd\nnew desc\n7\n")

	if store.todos[0].Description != "new desc" {
		t.Fatalf("expected 'new desc', got %q", store.todos[0].Description)
	}
	if !strings.Contains(output, "Description updated successfully.") {
		t.Fatalf("expected description update message in output, got:\n%s", output)
	}
}

func TestEditTitleUnchanged(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nsame\n\n6\n1\nt\nsame\n7\n")

	if !strings.Contains(output, "Info: title is already the same.") {
		t.Fatalf("expected info message, got:\n%s", output)
	}
}

func TestEditDescriptionUnchanged(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\ntask\nsame desc\n6\n1\nd\nsame desc\n7\n")

	if !strings.Contains(output, "Info: description is already the same.") {
		t.Fatalf("expected info message, got:\n%s", output)
	}
}

func TestEditBothStopsOnTitleError(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\nold title\nold desc\n6\n1\nb\n\n7\n")

	if !strings.Contains(output, "Error: title cannot be empty") {
		t.Fatalf("expected title error, got:\n%s", output)
	}
	if strings.Contains(output, "> Enter new description:") {
		t.Fatalf("should not prompt for description after title error, got:\n%s", output)
	}
}

func TestEditInvalidFieldChoice(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "1\ntask\n\n6\n1\nx\n7\n")

	if !strings.Contains(output, "Error: invalid choice") || !strings.Contains(output, "enter 't', 'd', or 'b'") {
		t.Fatalf("expected invalid choice error, got:\n%s", output)
	}
}

func TestInvalidMenuChoice(t *testing.T) {
	store := newMockStorage()
	output := runApp(t, store, "99\nabc\n7\n")

	if !strings.Contains(output, "Error: please enter a number between 0 and 7.") {
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
