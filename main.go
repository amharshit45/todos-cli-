package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const todosFile = "todos.json"

type Todo struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// --- Storage layer ---

type TodoStorage interface {
	Load() ([]Todo, error)
	Save(todos []Todo) error
}

type JSONStorage struct {
	filePath string
}

func NewJSONStorage(filePath string) *JSONStorage {
	return &JSONStorage{filePath: filePath}
}

func (js *JSONStorage) Load() ([]Todo, error) {
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
			return []Todo{}, nil
		}
		return nil, fmt.Errorf("unable to open todos file: %w", err)
	}
	defer file.Close()

	var todos []Todo
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&todos); err != nil {
		return nil, fmt.Errorf("unable to unmarshal todos file: %w", err)
	}
	return todos, nil
}

func (js *JSONStorage) Save(todos []Todo) error {
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

// --- Business logic layer ---

type TodoHandler interface {
	Add(description string)
	List() []Todo
	Delete(id int) error
	SetCompleted(id int) error
	SetIncomplete(id int) error
	Edit(id int, description string) error
	Save() error
}

type TodoStore struct {
	mu      sync.Mutex
	todos   []Todo
	storage TodoStorage
	maxID   int
}

func NewTodoStore(storage TodoStorage) (*TodoStore, error) {
	todos, err := storage.Load()
	if err != nil {
		return nil, err
	}

	maxID := 0
	for _, t := range todos {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	return &TodoStore{
		todos:   todos,
		storage: storage,
		maxID:   maxID,
	}, nil
}

func (s *TodoStore) nextID() int {
	s.maxID++
	return s.maxID
}

func (s *TodoStore) findByID(id int) (int, error) {
	if id <= 0 {
		return -1, fmt.Errorf("invalid id: %d", id)
	}
	for i := range s.todos {
		if s.todos[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("todo with id %d not found", id)
}

func (s *TodoStore) Add(description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = append(s.todos, Todo{ID: s.nextID(), Description: description})
}

func (s *TodoStore) List() []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

func (s *TodoStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	s.todos = append(s.todos[:idx], s.todos[idx+1:]...)
	return nil
}

func (s *TodoStore) SetCompleted(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	if s.todos[idx].Completed {
		return fmt.Errorf("todo %d is already completed", id)
	}
	s.todos[idx].Completed = true
	return nil
}

func (s *TodoStore) SetIncomplete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	if !s.todos[idx].Completed {
		return fmt.Errorf("todo %d is already incomplete", id)
	}
	s.todos[idx].Completed = false
	return nil
}

func (s *TodoStore) Edit(id int, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	s.todos[idx].Description = description
	return nil
}

func (s *TodoStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storage.Save(s.todos)
}

// --- UI helpers ---

func printMenu() {
	fmt.Println("===== Todo CLI =====")
	fmt.Println("1. Add a todo")
	fmt.Println("2. List todos")
	fmt.Println("3. Delete a todo")
	fmt.Println("4. Mark as completed")
	fmt.Println("5. Mark as incomplete")
	fmt.Println("6. Edit a todo")
	fmt.Println("7. Exit")
	fmt.Println("====================")
}

func printTodos(todos []Todo) {
	if len(todos) == 0 {
		fmt.Println("No todos found.")
		return
	}
	for _, todo := range todos {
		if todo.Completed {
			fmt.Printf("[✓] %d. \033[9m%s\033[0m\n", todo.ID, todo.Description)
		} else {
			fmt.Printf("[ ] %d. %s\n", todo.ID, todo.Description)
		}
	}
}

func readLine(scanner *bufio.Scanner, prompt string) (string, bool) {
	fmt.Print(prompt)
	if !scanner.Scan() {
		return "", false
	}
	return strings.TrimSpace(scanner.Text()), true
}

func readID(scanner *bufio.Scanner, prompt string) (int, bool, error) {
	input, ok := readLine(scanner, prompt)
	if !ok {
		return 0, false, nil
	}
	id, err := strconv.Atoi(input)
	if err != nil {
		return 0, true, fmt.Errorf("invalid ID: '%s' is not a number", input)
	}
	return id, true, nil
}

// --- Main ---

func main() {
	storage := NewJSONStorage(todosFile)
	store, err := NewTodoStore(storage)
	if err != nil {
		fmt.Printf("Error loading todos: %v\n", err)
		return
	}
	var handler TodoHandler = store

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if err := handler.Save(); err != nil {
			fmt.Printf("\nError saving todos during shutdown: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	printMenu()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		choice, ok := readLine(scanner, "> Choose an option: ")
		if !ok {
			break
		}

		option, err := strconv.Atoi(choice)
		if err != nil || option < 1 || option > 7 {
			fmt.Println("Error: please enter a number between 1 and 7.")
			continue
		}

		switch option {
		case 1: // Add
			desc, ok := readLine(scanner, "> Enter description: ")
			if !ok {
				return
			}
			if desc == "" {
				fmt.Println("Error: description cannot be empty.")
			} else {
				handler.Add(desc)
				if err := handler.Save(); err != nil {
					fmt.Printf("Warning: failed to save todos: %v\n", err)
				} else {
					fmt.Println("Todo added successfully.")
				}
			}

		case 2: // List
			printTodos(handler.List())

		case 3: // Delete
			printTodos(handler.List())
			id, ok, err := readID(scanner, "> Enter todo ID to delete: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.Delete(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.Save(); err != nil {
				fmt.Printf("Warning: failed to save todos: %v\n", err)
			} else {
				fmt.Println("Todo deleted successfully.")
			}

		case 4: // Mark as completed
			printTodos(handler.List())
			id, ok, err := readID(scanner, "> Enter todo ID to mark as completed: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.SetCompleted(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.Save(); err != nil {
				fmt.Printf("Warning: failed to save todos: %v\n", err)
			} else {
				fmt.Println("Todo marked as completed.")
			}

		case 5: // Mark as incomplete
			printTodos(handler.List())
			id, ok, err := readID(scanner, "> Enter todo ID to mark as incomplete: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.SetIncomplete(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.Save(); err != nil {
				fmt.Printf("Warning: failed to save todos: %v\n", err)
			} else {
				fmt.Println("Todo marked as incomplete.")
			}

		case 6: // Edit
			printTodos(handler.List())
			id, ok, err := readID(scanner, "> Enter todo ID to edit: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				desc, ok := readLine(scanner, "> Enter new description: ")
				if !ok {
					return
				}
				if desc == "" {
					fmt.Println("Error: description cannot be empty.")
				} else if err := handler.Edit(id, desc); err != nil {
					fmt.Printf("Error: %v\n", err)
				} else if err := handler.Save(); err != nil {
					fmt.Printf("Warning: failed to save todos: %v\n", err)
				} else {
					fmt.Println("Todo updated successfully.")
				}
			}

		case 7: // Exit
			if err := handler.Save(); err != nil {
				fmt.Printf("Error saving todos: %v\n", err)
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
