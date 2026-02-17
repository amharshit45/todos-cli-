package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Todo struct {
	ID          int
	Description string
	Completed   bool
}

/*
1. Read the JSON file
2. Take input from user for command and description
3. Perform the action based on the command
4. Save the JSON file
*/

func readTodos() ([]Todo, error) {
	todoBytes, err := os.ReadFile("todos.json")
	if err != nil {
		return nil, err
	}

	var todos []Todo
	err = json.Unmarshal(todoBytes, &todos)
	if err != nil {
		return nil, err
	}
	return todos, nil
}

func updateTodos(todos []Todo) error {
	todoBytes, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile("todos.json", todoBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func removeTodo(todos []Todo, id int) []Todo {
	for i, todo := range todos {
		if todo.ID == id {
			return append(todos[:i], todos[i+1:]...)
		}
	}
	return todos
}

func markComplete(todos []Todo, id int) []Todo {
	for i, todo := range todos {
		if todo.ID == id {
			todos[i].Completed = true
			return todos
		}
	}
	return todos
}

func printTodos(todos []Todo) {
	for _, todo := range todos {
		fmt.Printf("%d. %s\n", todo.ID, todo.Description)
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
			todos = removeTodo(todos, id)
		case "complete":
			fmt.Println("Enter the id of the todo to mark as complete:")
			var id int
			// TODO: Validate id is a number.
			fmt.Scanln(&id)
			todos = markComplete(todos, id)
		case "exit":
			return
		default:
			fmt.Println("Invalid command")
		}
		// TODO: Add a loop to update the todos file until it is successful, but also quit if there is a persistent error.
		updateTodos(todos)
	}
}
