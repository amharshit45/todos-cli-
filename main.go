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

// TODO: Consider using a map instead of a slice for the todos.

type Todo struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func parseCommand(line string) (string, []string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, fmt.Errorf("empty command")
	}

	var tokens []string
	var currentToken strings.Builder
	inQuotes := false
	escaped := false

	for i, char := range line {
		if escaped {
			currentToken.WriteRune(char)
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			if inQuotes {
				// End of quoted string
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
				inQuotes = false
			} else {
				// Start of quoted string
				if currentToken.Len() > 0 {
					return "", nil, fmt.Errorf("unexpected quote at position %d", i)
				}
				inQuotes = true
			}
			continue
		}

		if char == ' ' || char == '\t' {
			if inQuotes {
				currentToken.WriteRune(char)
			} else if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			continue
		}

		currentToken.WriteRune(char)
	}

	if inQuotes {
		return "", nil, fmt.Errorf("unclosed quote")
	}

	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	command := tokens[0]
	args := tokens[1:]

	return command, args, nil
}

// CommandSpec defines the specification for a command
type CommandSpec struct {
	NumArgs  int
	ArgTypes []string // "int" or "string"
}

var commandSpecs = map[string]CommandSpec{
    "add":        {NumArgs: 1, ArgTypes: []string{"string"}},
    "list":       {NumArgs: 0, ArgTypes: []string{}},
    "delete":     {NumArgs: 1, ArgTypes: []string{"int"}},
    "completed":  {NumArgs: 1, ArgTypes: []string{"int"}},
    "incomplete": {NumArgs: 1, ArgTypes: []string{"int"}},
    "edit":       {NumArgs: 2, ArgTypes: []string{"int", "string"}},
    "exit":       {NumArgs: 0, ArgTypes: []string{}},
}

// validateCommand validates a command and its arguments
func validateCommand(cmd string, args []string) error {
	spec, exists := commandSpecs[cmd]
	if !exists {
		return fmt.Errorf("invalid command: '%s'", cmd)
	}

	if len(args) != spec.NumArgs {
		return fmt.Errorf("command '%s' requires exactly %d argument(s)", cmd, spec.NumArgs)
	}

	for i, arg := range args {
		if i < len(spec.ArgTypes) {
			expectedType := spec.ArgTypes[i]
			if expectedType == "int" {
				if _, err := strconv.Atoi(arg); err != nil {
					return fmt.Errorf("invalid ID: '%s' is not a number", arg)
				}
			}
		}
	}

	return nil
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

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("> ")
		if !scanner.Scan() {
			// EOF or error
			break
		}

		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse command and arguments
		cmd, args, err := parseCommand(line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Validate command
		if err := validateCommand(cmd, args); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		todosMutex.Lock()

		switch cmd {
		case "add":
			description := args[0]
			todos = append(todos, Todo{ID: int(len(todos) + 1), Description: description})
			fmt.Println("Todo added successfully.")

		case "list":
			printTodos(todos)

		case "delete":
			id, _ := strconv.Atoi(args[0]) // Already validated
			todos, err = removeTodo(todos, id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo deleted successfully.")
			}

		case "completed":
			id, _ := strconv.Atoi(args[0]) // Already validated
			err = toggleComplete(todos, id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo marked as completed.")
			}

	case "incomplete":
		id, _ := strconv.Atoi(args[0]) // Already validated
		err = toggleComplete(todos, id)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Todo marked as incomplete.")
		}

	case "edit":
		id, _ := strconv.Atoi(args[0]) // Already validated
		newDescription := args[1]
		err = editTodo(todos, id, newDescription)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Todo updated successfully.")
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
			fmt.Printf("Error: invalid command: '%s'\n", cmd)
		}

		// Save todos after each command (except list which doesn't modify)
		if cmd != "list" && cmd != "exit" {
			if err := updateTodos(todos); err != nil {
				fmt.Printf("Warning: failed to save todos: %v\n", err)
			}
		}

		todosMutex.Unlock()
	}

	// Handle scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
