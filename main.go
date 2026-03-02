package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Todo struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

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
		return todos, fmt.Errorf("invalid id: %d", id)
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

func editTodo(todos []Todo, id int, newDescription string) error {
	if id <= 0 {
		return fmt.Errorf("Invalid id: %d", id)
	}
	for i, todo := range todos {
		if todo.ID == id {
			todos[i].Description = newDescription
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

func generateTodoID() int {
	return rand.Intn(1000)
}

func main() {
	todos, err := readTodos()
	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		return
	}
	var todosMutex sync.Mutex

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		todosMutex.Lock()
		defer todosMutex.Unlock()
		if err := updateTodos(todos); err != nil {
			fmt.Printf("Error saving todos during shutdown: %v\n", err)
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

		todosMutex.Lock()
		modified := false

		switch option {
		case 1: // Add
			desc, ok := readLine(scanner, "> Enter description: ")
			if !ok {
				todosMutex.Unlock()
				return
			}
			if desc == "" {
				fmt.Println("Error: description cannot be empty.")
			} else {
				todos = append(todos, Todo{ID: generateTodoID(), Description: desc})
				fmt.Println("Todo added successfully.")
				modified = true
			}

		case 2: // List
			printTodos(todos)

		case 3: // Delete
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to delete: ")
			if !ok {
				todosMutex.Unlock()
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				todos, err = removeTodo(todos, id)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println("Todo deleted successfully.")
					modified = true
				}
			}

		case 4: // Mark as completed
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to mark as completed: ")
			if !ok {
				todosMutex.Unlock()
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				err = toggleComplete(todos, id)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println("Todo marked as completed.")
					modified = true
				}
			}

		case 5: // Mark as incomplete
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to mark as incomplete: ")
			if !ok {
				todosMutex.Unlock()
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				err = toggleComplete(todos, id)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Println("Todo marked as incomplete.")
					modified = true
				}
			}

		case 6: // Edit
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to edit: ")
			if !ok {
				todosMutex.Unlock()
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				desc, ok := readLine(scanner, "> Enter new description: ")
				if !ok {
					todosMutex.Unlock()
					return
				}
				if desc == "" {
					fmt.Println("Error: description cannot be empty.")
				} else {
					err = editTodo(todos, id, desc)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					} else {
						fmt.Println("Todo updated successfully.")
						modified = true
					}
				}
			}

		case 7: // Exit
			if err := updateTodos(todos); err != nil {
				fmt.Printf("Error saving todos: %v\n", err)
			}
			todosMutex.Unlock()
			return
		}

		if modified {
			if err := updateTodos(todos); err != nil {
				fmt.Printf("Warning: failed to save todos: %v\n", err)
			}
		}

		todosMutex.Unlock()
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
