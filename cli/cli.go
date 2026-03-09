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
	ansiStrikethrough = "\033[9m"
	ansiReset         = "\033[0m"
)

var errExit = errors.New("exit requested")

type menuItem struct {
	label   string
	handler func(ctx context.Context) error
}

type App struct {
	store   todo.Storage
	scanner *bufio.Scanner
	menu    []menuItem
	lines   chan string
	scanErr error
}

func New(store todo.Storage, scanner *bufio.Scanner) *App {
	app := &App{
		store:   store,
		scanner: scanner,
		lines:   make(chan string),
	}
	app.menu = []menuItem{
		{"Add a todo", app.handleAdd},
		{"List todos", app.handleList},
		{"Delete a todo", app.handleDelete},
		{"Mark as completed", func(ctx context.Context) error { return app.handleSetCompleted(ctx, true) }},
		{"Mark as incomplete", func(ctx context.Context) error { return app.handleSetCompleted(ctx, false) }},
		{"Edit a todo", app.handleEdit},
		{"Exit", func(context.Context) error { return errExit }},
	}
	return app
}

func (a *App) readInput(ctx context.Context) {
	defer close(a.lines)
	for a.scanner.Scan() {
		select {
		case a.lines <- a.scanner.Text():
		case <-ctx.Done():
			return
		}
	}
	a.scanErr = a.scanner.Err()
}

func (a *App) Run(ctx context.Context) error {
	go a.readInput(ctx)
	a.printMenu()
	for {
		choice, err := a.readLine(ctx, "> Choose an option: ")
		if err != nil {
			if errors.Is(err, errExit) {
				return nil
			}
			return err
		}
		option, parseErr := strconv.Atoi(choice)
		if parseErr != nil || option < 1 || option > len(a.menu) {
			fmt.Printf("Error: please enter a number between 1 and %d.\n", len(a.menu))
			continue
		}
		if err := a.menu[option-1].handler(ctx); err != nil {
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

func (a *App) readLine(ctx context.Context, prompt string) (string, error) {
	fmt.Print(prompt)
	select {
	case <-ctx.Done():
		return "", errExit
	case line, ok := <-a.lines:
		if !ok {
			if a.scanErr != nil {
				return "", fmt.Errorf("input error: %w", a.scanErr)
			}
			return "", errExit
		}
		return strings.TrimSpace(line), nil
	}
}

func (a *App) readID(ctx context.Context, prompt string) (int, error) {
	input, err := a.readLine(ctx, prompt)
	if err != nil {
		return 0, err
	}
	id, parseErr := strconv.Atoi(input)
	if parseErr != nil {
		return 0, fmt.Errorf("invalid ID %q: %w", input, parseErr)
	}
	return id, nil
}

func (a *App) listAndPromptID(ctx context.Context, prompt string) (int, error) {
	todos, err := a.store.List(ctx)
	if err != nil {
		return 0, err
	}
	printTodos(todos)
	return a.readID(ctx, prompt)
}

func (a *App) readDescription(ctx context.Context, prompt string) (string, error) {
	desc, err := a.readLine(ctx, prompt)
	if err != nil {
		return "", err
	}
	if err := todo.ValidateDescription(desc); err != nil {
		return "", err
	}
	return desc, nil
}

func (a *App) handleErr(err error) error {
	if errors.Is(err, errExit) {
		return err
	}
	fmt.Printf("Error: %v\n", err)
	return nil
}

func (a *App) handleAdd(ctx context.Context) error {
	desc, err := a.readDescription(ctx, "> Enter description: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Add(ctx, desc); err != nil {
		return a.handleErr(err)
	}
	fmt.Println("Todo added successfully.")
	return nil
}

func (a *App) handleList(ctx context.Context) error {
	todos, err := a.store.List(ctx)
	if err != nil {
		return a.handleErr(err)
	}
	printTodos(todos)
	return nil
}

func (a *App) handleDelete(ctx context.Context) error {
	id, err := a.listAndPromptID(ctx, "> Enter todo ID to delete: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Delete(ctx, id); err != nil {
		return a.handleErr(err)
	}
	fmt.Println("Todo deleted successfully.")
	return nil
}

func (a *App) handleSetCompleted(ctx context.Context, completed bool) error {
	action := "completed"
	if !completed {
		action = "incomplete"
	}
	id, err := a.listAndPromptID(ctx, fmt.Sprintf("> Enter todo ID to mark as %s: ", action))
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.SetCompleted(ctx, id, completed); err != nil {
		return a.handleErr(err)
	}
	fmt.Printf("Todo marked as %s.\n", action)
	return nil
}

func (a *App) handleEdit(ctx context.Context) error {
	id, err := a.listAndPromptID(ctx, "> Enter todo ID to edit: ")
	if err != nil {
		return a.handleErr(err)
	}
	desc, err := a.readDescription(ctx, "> Enter new description: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Edit(ctx, id, desc); err != nil {
		return a.handleErr(err)
	}
	fmt.Println("Todo updated successfully.")
	return nil
}
