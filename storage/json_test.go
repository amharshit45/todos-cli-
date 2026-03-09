package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/amharshit45/todos-cli-/todo"
)

func newTestStorage(t *testing.T) *JSONStorage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "todos.json")
	s, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	return s
}

func TestAddAndList(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	todos, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}

	if err := s.Add(ctx, "first"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(ctx, "second"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	todos, err = s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(todos))
	}
	if todos[0].Description != "first" || todos[1].Description != "second" {
		t.Fatalf("unexpected descriptions: %v", todos)
	}
	if todos[0].ID != 1 || todos[1].ID != 2 {
		t.Fatalf("unexpected IDs: %d, %d", todos[0].ID, todos[1].ID)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	s.Add(ctx, "to delete")
	s.Add(ctx, "to keep")

	if err := s.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	todos, _ := s.List(ctx)
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Description != "to keep" {
		t.Fatalf("wrong todo remaining: %v", todos[0])
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	err := s.Delete(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestDeleteInvalidID(t *testing.T) {
	s := newTestStorage(t)
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

func TestSetCompleted(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	s.Add(ctx, "task")

	if err := s.SetCompleted(ctx, 1, true); err != nil {
		t.Fatalf("SetCompleted(true): %v", err)
	}
	todos, _ := s.List(ctx)
	if !todos[0].Completed {
		t.Fatal("expected completed=true")
	}

	if err := s.SetCompleted(ctx, 1, false); err != nil {
		t.Fatalf("SetCompleted(false): %v", err)
	}
	todos, _ = s.List(ctx)
	if todos[0].Completed {
		t.Fatal("expected completed=false")
	}
}

func TestSetCompletedAlready(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	s.Add(ctx, "task")

	err := s.SetCompleted(ctx, 1, false)
	if err == nil {
		t.Fatal("expected error for already incomplete")
	}
	if !errors.Is(err, todo.ErrAlreadyIncomplete) {
		t.Fatalf("expected ErrAlreadyIncomplete, got: %v", err)
	}

	s.SetCompleted(ctx, 1, true)

	err = s.SetCompleted(ctx, 1, true)
	if err == nil {
		t.Fatal("expected error for already completed")
	}
	if !errors.Is(err, todo.ErrAlreadyCompleted) {
		t.Fatalf("expected ErrAlreadyCompleted, got: %v", err)
	}
}

func TestSetCompletedNotFound(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	err := s.SetCompleted(ctx, 999, true)
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestEdit(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	s.Add(ctx, "original")

	if err := s.Edit(ctx, 1, "updated"); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	todos, _ := s.List(ctx)
	if todos[0].Description != "updated" {
		t.Fatalf("expected 'updated', got '%s'", todos[0].Description)
	}
}

func TestEditNotFound(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	err := s.Edit(ctx, 999, "nope")
	if !errors.Is(err, todo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestClose(t *testing.T) {
	s := newTestStorage(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todos.json")

	s1, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}
	ctx := context.Background()

	s1.Add(ctx, "persistent")
	s1.Close()

	s2, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage (reopen): %v", err)
	}

	todos, err := s2.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 1 || todos[0].Description != "persistent" {
		t.Fatalf("expected 'persistent', got: %v", todos)
	}

	s2.Add(ctx, "second")
	todos, _ = s2.List(ctx)
	if todos[1].ID != 2 {
		t.Fatalf("expected ID 2 for second todo, got %d", todos[1].ID)
	}
}

func TestNewJSONStorageCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.json")

	s, err := NewJSONStorage(path)
	if err != nil {
		t.Fatalf("NewJSONStorage: %v", err)
	}

	todos, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected empty list, got %d", len(todos))
	}
}
