package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/amharshit45/todos-cli-/todo"
)

var _ todo.Storage = (*JSONStorage)(nil)

type JSONStorage struct {
	mu       sync.RWMutex
	filePath string
	maxID    int
}

func NewJSONStorage(filePath string) (*JSONStorage, error) {
	js := &JSONStorage{filePath: filePath}

	todos, err := js.load()
	if err != nil {
		return nil, err
	}

	for _, t := range todos {
		if t.ID > js.maxID {
			js.maxID = t.ID
		}
	}

	return js, nil
}

func (js *JSONStorage) load() ([]todo.Todo, error) {
	file, err := os.Open(js.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			f, createErr := os.Create(js.filePath)
			if createErr != nil {
				return nil, fmt.Errorf("unable to create todos file: %w", createErr)
			}
			if _, writeErr := f.Write([]byte("[]\n")); writeErr != nil {
				f.Close()
				return nil, fmt.Errorf("unable to write todos file: %w", writeErr)
			}
			if err := f.Close(); err != nil {
				return nil, fmt.Errorf("unable to close todos file: %w", err)
			}
			return []todo.Todo{}, nil
		}
		return nil, fmt.Errorf("unable to open todos file: %w", err)
	}
	defer file.Close()

	var todos []todo.Todo
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&todos); err != nil {
		return nil, fmt.Errorf("unable to unmarshal todos file: %w", err)
	}
	return todos, nil
}

func (js *JSONStorage) save(todos []todo.Todo) error {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal todos: %w", err)
	}
	data = append(data, '\n')
	tmpFile := js.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tmpFile, js.filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename file: %w", err)
	}
	return nil
}

func (js *JSONStorage) findByID(todos []todo.Todo, id int) (int, error) {
	if err := todo.ValidateID(id); err != nil {
		return -1, err
	}
	for i := range todos {
		if todos[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("todo with id %d: %w", id, todo.ErrNotFound)
}

func (js *JSONStorage) Add(ctx context.Context, description string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	js.mu.Lock()
	defer js.mu.Unlock()

	todos, err := js.load()
	if err != nil {
		return err
	}

	js.maxID++
	todos = append(todos, todo.Todo{ID: js.maxID, Description: description})
	return js.save(todos)
}

func (js *JSONStorage) List(ctx context.Context) ([]todo.Todo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	js.mu.RLock()
	defer js.mu.RUnlock()
	return js.load()
}

func (js *JSONStorage) Delete(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	js.mu.Lock()
	defer js.mu.Unlock()

	todos, err := js.load()
	if err != nil {
		return err
	}
	idx, err := js.findByID(todos, id)
	if err != nil {
		return err
	}
	todos = append(todos[:idx], todos[idx+1:]...)
	return js.save(todos)
}

func (js *JSONStorage) SetCompleted(ctx context.Context, id int, completed bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	js.mu.Lock()
	defer js.mu.Unlock()

	todos, err := js.load()
	if err != nil {
		return err
	}
	idx, err := js.findByID(todos, id)
	if err != nil {
		return err
	}
	if todos[idx].Completed == completed {
		if completed {
			return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyCompleted)
		}
		return fmt.Errorf("todo %d: %w", id, todo.ErrAlreadyIncomplete)
	}
	todos[idx].Completed = completed
	return js.save(todos)
}

func (js *JSONStorage) Edit(ctx context.Context, id int, description string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := todo.ValidateDescription(description); err != nil {
		return err
	}
	js.mu.Lock()
	defer js.mu.Unlock()

	todos, err := js.load()
	if err != nil {
		return err
	}
	idx, err := js.findByID(todos, id)
	if err != nil {
		return err
	}
	todos[idx].Description = description
	return js.save(todos)
}

func (js *JSONStorage) Close(_ context.Context) error {
	return nil
}
