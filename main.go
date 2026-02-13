package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Todo struct {
	Id          int32
	Description string
	Completed   bool
}

/*
1. Read the JSON file
2. Take input from user for command and ?.description
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
	todoBytes, err := json.Marshal(todos)
	if err != nil {
		return err
	}
	os.WriteFile("todos.json", todoBytes, 0644)
	return nil
}


func removeTodo(todos []Todo, id int32) []Todo {
	for i, todo := range todos {
		if todo.Id == id {
			return append(todos[:i], todos[i+1:]...)
		}
	}
	return todos
}

func markComplete(todos []Todo, id int32) []Todo {
	for i, todo := range todos {
		if todo.Id == id {
			todos[i].Completed = true
			return todos
		}
	}
	return todos
}

func main() {
	todos, err := readTodos()
	if err != nil {
		fmt.Println("Error reading JSON file.")
	}
	command := os.Args[1]
	switch command {
	case "add":
		description := os.Args[2]
		todos = append(todos, Todo{Id: int32(len(todos) + 1), Description: description})
		// fmt.Println(todos)
	case "list":
		fmt.Println(todos)
	case "delete":
		id := os.Args[2]
		idInt, err := strconv.Atoi(id)
		if err != nil {
			fmt.Println("Invalid id")
			return
		}
		todos = removeTodo(todos, int32(idInt))
		// fmt.Println(todos)
	case "mark-complete":
		id := os.Args[2]
		idInt, err := strconv.Atoi(id)
		if err != nil {
			fmt.Println("Invalid id")
			return
		}
		todos = markComplete(todos, int32(idInt))
		// fmt.Println(todos)
	default:
		fmt.Println("Invalid command")
	}
	updateTodos(todos)
}
