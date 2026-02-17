package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// TODO: Consider using a map instead of a slice for the todos.

type Todo struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func readTodos() ([]Todo, error) {
	var todos []Todo
	file, err := os.Open("todos.json")
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create("todos.json")
			if err != nil {
				return nil, fmt.Errorf("Unable to create todos file: %w", err)
			}
			file.Write([]byte("[]"))
			file.Close()
			return []Todo{}, nil
		}
		return nil, fmt.Errorf("Unable to open todos file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&todos)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshal todos file: %w", err)
	}
	return todos, nil
}

func updateTodos(todos []Todo) error {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to marshal todos: %w", err)
	}
	data = append(data, '\n')
	tmpFile := "todos.json.tmp"
    if err := os.WriteFile(tmpFile, data, 0644); err != nil {
        return fmt.Errorf("failed to write temp file: %w", err)
    }
    if err := os.Rename(tmpFile, "todos.json"); err != nil {
        os.Remove(tmpFile) 
        return fmt.Errorf("failed to rename file: %w", err)
    }
    return nil
}

func removeTodo(todos []Todo, id int) ([]Todo, error) {
	if id <= 0 {
		return todos, fmt.Errorf("Invalid id: %d", id)
	}
	for i, todo := range todos {
		if todo.ID == id {
			return append(todos[:i], todos[i+1:]...), nil
		}
	}
	return todos, fmt.Errorf("Todo with id %d not found", id)
}

func toggleComplete(todos []Todo, id int) error {
	if id <= 0 {
		return fmt.Errorf("Invalid id: %d", id)
	}
	for i, todo := range todos {
		if todo.ID == id {
			todos[i].Completed = !todos[i].Completed
			return nil
		}
	}
	return fmt.Errorf("Todo with id %d not found", id)
}

func printTodos(todos []Todo) {
	if len(todos) == 0 {
		fmt.Println("No todos found.")
		return
	}
	for _, todo := range todos {
		if todo.Completed {
			// ANSI strikethrough: \033[9m text \033[0m
			fmt.Printf("[✓] %d. \033[9m%s\033[0m\n", todo.ID, todo.Description)
		} else {
			fmt.Printf("[ ] %d. %s\n", todo.ID, todo.Description)
		}
	}
}

func main() {
	todos, err := readTodos()
	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		return
	}

	// Mutex to protect todos from concurrent access
	var todosMutex sync.Mutex

	// Set up channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown on signal
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal, saving todos...")
		todosMutex.Lock()
		defer todosMutex.Unlock()
		if err := updateTodos(todos); err != nil {
			fmt.Printf("Error saving todos during shutdown: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Todos saved successfully. Goodbye!")
		os.Exit(0)
	}()

	for {
		fmt.Printf("> ")
		// TODO: Read the command as a complete line, not just a single word.
		// TODO: Use a scanner to read the command.
		// TODO: Validate commands.
		var command string
		fmt.Scanln(&command)
		
		todosMutex.Lock()
		switch command {
		case "add":
			fmt.Println("Enter a todo:")
			// TODO: Read mutliword description from user.
			var description string
			fmt.Scanln(&description)
			todos = append(todos, Todo{ID: int(len(todos) + 1), Description: description})
		case "list":
			printTodos(todos)
		case "delete":
			fmt.Println("Enter the id of the todo to delete:")
			var id int
			// TODO: Validate id is a number.
			fmt.Scanln(&id)
			todos, err = removeTodo(todos, id)
			if err != nil {
				fmt.Printf("Error deleting todo: %v\n", err)
			}
		case "complete":
			fmt.Println("Enter the id of the todo to mark as complete:")
			var id int
			// TODO: Validate id is a number.
			fmt.Scanln(&id)
			err = toggleComplete(todos, id)
			if err != nil {
				fmt.Printf("Error marking todo as complete: %v\n", err)
			}
		case "incomplete":
			fmt.Println("Enter the id of the todo to mark as incomplete:")
			var id int
			// TODO: Validate id is a number.
			fmt.Scanln(&id)
			err = toggleComplete(todos, id)
			if err != nil {
				fmt.Printf("Error marking todo as incomplete: %v\n", err)
			}
		case "exit":
			if err := updateTodos(todos); err != nil {
				fmt.Printf("Error saving todos: %v\n", err)
				todosMutex.Unlock()
				return
			}
			todosMutex.Unlock()
			return
		default:
			fmt.Println("Invalid command")
		}
		todosMutex.Unlock()
	}
}
