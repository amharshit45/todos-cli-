package main

import (
	"encoding/json"
	"fmt"
	"os"
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

// TODO: Make the writes atomic.
// TODO: Consider backing up the file before writing.
// TODO: Consider using checksums to verify the file integrity.
func updateTodos(todos []Todo) error {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to marshal todos: %w", err)
	}
	data = append(data, '\n')
	err = os.WriteFile("todos.json", data, 0644)
	if err != nil {
		return fmt.Errorf("Unable to write to todos file: %w", err)
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
	for {
		fmt.Printf("> ")
		// TODO: Read the command as a complete line, not just a single word.
		// TODO: Use a scanner to read the command.
		// TODO: Validate commands.
		var command string
		fmt.Scanln(&command)
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
			return
		default:
			fmt.Println("Invalid command")
		}
		// TODO: Add a loop to update the todos file until it is successful, but also quit if there is a persistent error.
		updateTodos(todos)
	}
}

// TODO: Add graceful shutdown incase of interrupt signal.
