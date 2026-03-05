package main

import (
	"bufio"
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
	var handler todo.TodoStorage = store

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		store.Close()
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
			} else if err := handler.Add(desc); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo added successfully.")
			}

		case 2: // List
			todos, err := handler.List()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				printTodos(todos)
			}

		case 3: // Delete
			todos, err := handler.List()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to delete: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.Delete(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo deleted successfully.")
			}

		case 4: // Mark as completed
			todos, err := handler.List()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to mark as completed: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.SetCompleted(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo marked as completed.")
			}

		case 5: // Mark as incomplete
			todos, err := handler.List()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			printTodos(todos)
			id, ok, err := readID(scanner, "> Enter todo ID to mark as incomplete: ")
			if !ok {
				return
			}
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else if err := handler.SetIncomplete(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Todo marked as incomplete.")
			}

		case 6: // Edit
			todos, err := handler.List()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			printTodos(todos)
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
				} else {
					fmt.Println("Todo updated successfully.")
				}
			}

		case 7: // Exit
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}
