package storage

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/amharshit45/todos-cli-/todo"
)

func newTestMongoStorage(t *testing.T) *MongoStorage {
	t.Helper()

	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set, skipping MongoDB tests")
	}

	dbName := "test_todos_cli"
	s, err := NewMongoStorage(context.Background(), uri, dbName)
	if err != nil {
		t.Fatalf("NewMongoStorage: %v", err)
	}

	ctx := context.Background()
	s.client.Database(dbName).Collection(collectionName).Drop(ctx)
	s.client.Database(dbName).Collection(counterCollection).Drop(ctx)

	t.Cleanup(func() {
		s.client.Database(dbName).Drop(context.Background())
		s.Close(context.Background())
	})

	return s
}

func TestMongoAddAndList(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}

	if err := s.Add(ctx, "first", "first details"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(ctx, "second", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	todos, err = s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(todos))
	}
	if todos[0].Title != "first" || todos[0].Description != "first details" {
		t.Fatalf("unexpected first todo: %+v", todos[0])
	}
	if todos[1].Title != "second" || todos[1].Description != "" {
		t.Fatalf("unexpected second todo: %+v", todos[1])
	}
	if todos[0].ID != 1 || todos[1].ID != 2 {
		t.Fatalf("unexpected IDs: %d, %d", todos[0].ID, todos[1].ID)
	}
}

func TestMongoAddEmptyTitle(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.Add(ctx, "", "some desc")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !errors.Is(err, todo.ErrEmptyTitle) {
		t.Fatalf("expected ErrEmptyTitle, got: %v", err)
	}
}

func TestMongoDelete(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "to delete", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(ctx, "to keep", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "to keep" {
		t.Fatalf("wrong todo remaining: %v", todos[0])
	}
}

func TestMongoDeleteNotFound(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.Delete(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestMongoDeleteInvalidID(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.Delete(ctx, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, todo.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got: %v", err)
	}

	err = s.Delete(ctx, -1)
	if !errors.Is(err, todo.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID for -1, got: %v", err)
	}
}

func TestMongoSetCompleted(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.SetCompleted(ctx, 1, true); err != nil {
		t.Fatalf("SetCompleted(true): %v", err)
	}
	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !todos[0].Completed {
		t.Fatal("expected completed=true")
	}

	if err := s.SetCompleted(ctx, 1, false); err != nil {
		t.Fatalf("SetCompleted(false): %v", err)
	}
	todos, err = s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if todos[0].Completed {
		t.Fatal("expected completed=false")
	}
}

func TestMongoSetCompletedAlready(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := s.SetCompleted(ctx, 1, false)
	if err == nil {
		t.Fatal("expected error for already incomplete")
	}
	if !errors.Is(err, todo.ErrAlreadyIncomplete) {
		t.Fatalf("expected ErrAlreadyIncomplete, got: %v", err)
	}

	if err := s.SetCompleted(ctx, 1, true); err != nil {
		t.Fatalf("SetCompleted(true): %v", err)
	}

	err = s.SetCompleted(ctx, 1, true)
	if err == nil {
		t.Fatal("expected error for already completed")
	}
	if !errors.Is(err, todo.ErrAlreadyCompleted) {
		t.Fatalf("expected ErrAlreadyCompleted, got: %v", err)
	}
}

func TestMongoSetCompletedNotFound(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.SetCompleted(ctx, 999, true)
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestMongoEditTitle(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "original", "desc"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.EditTitle(ctx, 1, "updated"); err != nil {
		t.Fatalf("EditTitle: %v", err)
	}
	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if todos[0].Title != "updated" {
		t.Fatalf("expected 'updated', got '%s'", todos[0].Title)
	}
	if todos[0].Description != "desc" {
		t.Fatalf("description should be unchanged, got '%s'", todos[0].Description)
	}
}

func TestMongoEditTitleNotFound(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.EditTitle(ctx, 999, "nope")
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestMongoEditTitleUnchanged(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "same", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := s.EditTitle(ctx, 1, "same")
	if !errors.Is(err, todo.ErrTitleUnchanged) {
		t.Fatalf("expected ErrTitleUnchanged, got: %v", err)
	}
}

func TestMongoEditTitleEmpty(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := s.EditTitle(ctx, 1, "")
	if !errors.Is(err, todo.ErrEmptyTitle) {
		t.Fatalf("expected ErrEmptyTitle, got: %v", err)
	}
}

func TestMongoEditDescription(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", "old"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.EditDescription(ctx, 1, "new"); err != nil {
		t.Fatalf("EditDescription: %v", err)
	}
	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if todos[0].Description != "new" {
		t.Fatalf("expected 'new', got '%s'", todos[0].Description)
	}
	if todos[0].Title != "task" {
		t.Fatalf("title should be unchanged, got '%s'", todos[0].Title)
	}
}

func TestMongoEditDescriptionNotFound(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	err := s.EditDescription(ctx, 999, "nope")
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestMongoEditDescriptionUnchanged(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", "same"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	err := s.EditDescription(ctx, 1, "same")
	if !errors.Is(err, todo.ErrDescriptionUnchanged) {
		t.Fatalf("expected ErrDescriptionUnchanged, got: %v", err)
	}
}

func TestMongoEditDescriptionToEmpty(t *testing.T) {
	s := newTestMongoStorage(t)
	ctx := context.Background()

	if err := s.Add(ctx, "task", "has desc"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.EditDescription(ctx, 1, ""); err != nil {
		t.Fatalf("EditDescription to empty: %v", err)
	}
	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if todos[0].Description != "" {
		t.Fatalf("expected empty description, got %q", todos[0].Description)
	}
}

func TestMongoClose(t *testing.T) {
	s := newTestMongoStorage(t)
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
