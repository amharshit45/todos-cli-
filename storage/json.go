package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"github.com/amharshit45/todos-cli-/todo"
)

type JSONStorage struct {
	filePath string
}

func NewJSONStorage(filePath string) *JSONStorage {
	return &JSONStorage{filePath: filePath}
}

func (js *JSONStorage) Load() ([]todo.Todo, error) {
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

func (js *JSONStorage) Save(todos []todo.Todo) error {
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
