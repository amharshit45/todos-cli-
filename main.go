package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/amharshit45/todos-cli-/storage"
	"github.com/amharshit45/todos-cli-/todo"
)

const menuOptions = 7

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

func printTodos(todos []todo.Todo) {
	if len(todos) == 0 {
		fmt.Println("No todos found.")
		return
	}
	for _, t := range todos {
		if t.Completed {
			fmt.Printf("[✓] %d. \033[9m%s\033[0m\n", t.ID, t.Description)
		} else {
			fmt.Printf("[ ] %d. %s\n", t.ID, t.Description)
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

// listAndPromptID lists all todos, prints them, and prompts for an ID selection.
// ok=false means the scanner hit EOF and the caller should exit.
func listAndPromptID(ctx context.Context, store todo.Storage, scanner *bufio.Scanner, prompt string) (int, bool, error) {
	todos, err := store.List(ctx)
	if err != nil {
		return 0, true, err
	}
	printTodos(todos)
	return readID(scanner, prompt)
}

func handleAdd(ctx context.Context, store todo.Storage, scanner *bufio.Scanner) bool {
	desc, ok := readLine(scanner, "> Enter description: ")
	if !ok {
		return false
	}
	if desc == "" {
		fmt.Println("Error: description cannot be empty.")
		return true
	}
	if err := store.Add(ctx, desc); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Todo added successfully.")
	}
	return true
}

func handleList(ctx context.Context, store todo.Storage) {
	todos, err := store.List(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printTodos(todos)
}

func handleDelete(ctx context.Context, store todo.Storage, scanner *bufio.Scanner) bool {
	id, ok, err := listAndPromptID(ctx, store, scanner, "> Enter todo ID to delete: ")
	if !ok {
		return false
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return true
	}
	if err := store.Delete(ctx, id); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Todo deleted successfully.")
	}
	return true
}

func handleSetCompleted(ctx context.Context, store todo.Storage, scanner *bufio.Scanner, completed bool) bool {
	action := "completed"
	if !completed {
		action = "incomplete"
	}
	id, ok, err := listAndPromptID(ctx, store, scanner, fmt.Sprintf("> Enter todo ID to mark as %s: ", action))
	if !ok {
		return false
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return true
	}
	if err := store.SetCompleted(ctx, id, completed); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Todo marked as %s.\n", action)
	}
	return true
}

func handleEdit(ctx context.Context, store todo.Storage, scanner *bufio.Scanner) bool {
	id, ok, err := listAndPromptID(ctx, store, scanner, "> Enter todo ID to edit: ")
	if !ok {
		return false
	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return true
	}
	desc, ok := readLine(scanner, "> Enter new description: ")
	if !ok {
		return false
	}
	if desc == "" {
		fmt.Println("Error: description cannot be empty.")
		return true
	}
	if err := store.Edit(ctx, id, desc); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Todo updated successfully.")
	}
	return true
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")
	if mongoURI == "" || mongoDB == "" {
		log.Fatal("MONGO_URI and MONGO_DB must be set in .env")
	}

	store, err := storage.NewMongoStorage(mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer store.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		os.Stdin.Close()
	}()

	printMenu()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		choice, ok := readLine(scanner, "> Choose an option: ")
		if !ok {
			break
		}

		option, err := strconv.Atoi(choice)
		if err != nil || option < 1 || option > menuOptions {
			fmt.Printf("Error: please enter a number between 1 and %d.\n", menuOptions)
			continue
		}

		switch option {
		case 1:
			if !handleAdd(ctx, store, scanner) {
				return
			}
		case 2:
			handleList(ctx, store)
		case 3:
			if !handleDelete(ctx, store, scanner) {
				return
			}
		case 4:
			if !handleSetCompleted(ctx, store, scanner, true) {
				return
			}
		case 5:
			if !handleSetCompleted(ctx, store, scanner, false) {
				return
			}
		case 6:
			if !handleEdit(ctx, store, scanner) {
				return
			}
		case 7:
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
