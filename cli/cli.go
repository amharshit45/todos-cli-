package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/amharshit45/todos-cli-/todo"
)

const (
	maxDescriptionLength = 500

	ansiStrikethrough = "\033[9m"
	ansiReset         = "\033[0m"
)

var errExit = errors.New("exit requested")

type menuItem struct {
	label   string
	handler func() error
}

type App struct {
	store   todo.Storage
	scanner *bufio.Scanner
	ctx     context.Context
	menu    []menuItem
}

func New(ctx context.Context, store todo.Storage, scanner *bufio.Scanner) *App {
	app := &App{
		store:   store,
		scanner: scanner,
		ctx:     ctx,
	}
	app.menu = []menuItem{
		{"Add a todo", app.handleAdd},
		{"List todos", app.handleList},
		{"Delete a todo", app.handleDelete},
		{"Mark as completed", func() error { return app.handleSetCompleted(true) }},
		{"Mark as incomplete", func() error { return app.handleSetCompleted(false) }},
		{"Edit a todo", app.handleEdit},
		{"Exit", func() error { return errExit }},
	}
	return app
}

func (a *App) Run() error {
	a.printMenu()
	for {
		choice, err := a.readLine("> Choose an option: ")
		if err != nil {
			return nil
		}
		option, parseErr := strconv.Atoi(choice)
		if parseErr != nil || option < 1 || option > len(a.menu) {
			fmt.Printf("Error: please enter a number between 1 and %d.\n", len(a.menu))
			continue
		}
		if err := a.menu[option-1].handler(); err != nil {
			if errors.Is(err, errExit) {
				return nil
			}
			return err
		}
	}
}

func (a *App) printMenu() {
	fmt.Println("===== Todo CLI =====")
	for i, item := range a.menu {
		fmt.Printf("%d. %s\n", i+1, item.label)
	}
	fmt.Println("====================")
}

func printTodos(todos []todo.Todo) {
	if len(todos) == 0 {
		fmt.Println("No todos found.")
		return
	}
	for _, t := range todos {
		if t.Completed {
			fmt.Printf("[✓] %d. %s%s%s\n", t.ID, ansiStrikethrough, t.Description, ansiReset)
		} else {
			fmt.Printf("[ ] %d. %s\n", t.ID, t.Description)
		}
	}
}

func (a *App) readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	if !a.scanner.Scan() {
		return "", errExit
	}
	return strings.TrimSpace(a.scanner.Text()), nil
}

func (a *App) readID(prompt string) (int, error) {
	input, err := a.readLine(prompt)
	if err != nil {
		return 0, err
	}
	id, parseErr := strconv.Atoi(input)
	if parseErr != nil {
		return 0, fmt.Errorf("invalid ID: '%s' is not a number", input)
	}
	return id, nil
}

func (a *App) listAndPromptID(prompt string) (int, error) {
	todos, err := a.store.List(a.ctx)
	if err != nil {
		return 0, err
	}
	printTodos(todos)
	return a.readID(prompt)
}

func (a *App) readDescription(prompt string) (string, error) {
	desc, err := a.readLine(prompt)
	if err != nil {
		return "", err
	}
	if desc == "" {
		return "", fmt.Errorf("description cannot be empty")
	}
	if len(desc) > maxDescriptionLength {
		return "", fmt.Errorf("description exceeds maximum length of %d characters", maxDescriptionLength)
	}
	return desc, nil
}

func (a *App) handleAdd() error {
	desc, err := a.readDescription("> Enter description: ")
	if err != nil {
		if errors.Is(err, errExit) {
			return err
		}
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	if err := a.store.Add(a.ctx, desc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Println("Todo added successfully.")
	return nil
}

func (a *App) handleList() error {
	todos, err := a.store.List(a.ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	printTodos(todos)
	return nil
}

func (a *App) handleDelete() error {
	id, err := a.listAndPromptID("> Enter todo ID to delete: ")
	if err != nil {
		if errors.Is(err, errExit) {
			return err
		}
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	if err := a.store.Delete(a.ctx, id); err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Println("Todo deleted successfully.")
	return nil
}

func (a *App) handleSetCompleted(completed bool) error {
	action := "completed"
	if !completed {
		action = "incomplete"
	}
	id, err := a.listAndPromptID(fmt.Sprintf("> Enter todo ID to mark as %s: ", action))
	if err != nil {
		if errors.Is(err, errExit) {
			return err
		}
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	if err := a.store.SetCompleted(a.ctx, id, completed); err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Printf("Todo marked as %s.\n", action)
	return nil
}

func (a *App) handleEdit() error {
	id, err := a.listAndPromptID("> Enter todo ID to edit: ")
	if err != nil {
		if errors.Is(err, errExit) {
			return err
		}
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	desc, err := a.readDescription("> Enter new description: ")
	if err != nil {
		if errors.Is(err, errExit) {
			return err
		}
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	if err := a.store.Edit(a.ctx, id, desc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil
	}
	fmt.Println("Todo updated successfully.")
	return nil
}
