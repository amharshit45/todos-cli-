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
	handler func() error
}

type App struct {
	store    todo.Storage
	ctx      context.Context
	menu     []menuItem
	lines    chan string
	scanDone chan struct{}
	scanErr  error
}

func New(ctx context.Context, store todo.Storage, scanner *bufio.Scanner) *App {
	app := &App{
		store:    store,
		ctx:      ctx,
		lines:    make(chan string),
		scanDone: make(chan struct{}),
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
	go app.readInput(scanner)
	return app
}

func (a *App) readInput(scanner *bufio.Scanner) {
	defer close(a.scanDone)
	defer close(a.lines)
	for scanner.Scan() {
		select {
		case a.lines <- scanner.Text():
		case <-a.ctx.Done():
			return
		}
	}
	a.scanErr = scanner.Err()
}

func (a *App) Run() error {
	a.printMenu()
	for {
		choice, err := a.readLine("> Choose an option: ")
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
	select {
	case <-a.ctx.Done():
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
	if len(desc) > todo.MaxDescriptionLength {
		return "", fmt.Errorf("description exceeds maximum length of %d characters", todo.MaxDescriptionLength)
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

func (a *App) handleAdd() error {
	desc, err := a.readDescription("> Enter description: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Add(a.ctx, desc); err != nil {
		return a.handleErr(err)
	}
	fmt.Println("Todo added successfully.")
	return nil
}

func (a *App) handleList() error {
	todos, err := a.store.List(a.ctx)
	if err != nil {
		return a.handleErr(err)
	}
	printTodos(todos)
	return nil
}

func (a *App) handleDelete() error {
	id, err := a.listAndPromptID("> Enter todo ID to delete: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Delete(a.ctx, id); err != nil {
		return a.handleErr(err)
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
		return a.handleErr(err)
	}
	if err := a.store.SetCompleted(a.ctx, id, completed); err != nil {
		return a.handleErr(err)
	}
	fmt.Printf("Todo marked as %s.\n", action)
	return nil
}

func (a *App) handleEdit() error {
	id, err := a.listAndPromptID("> Enter todo ID to edit: ")
	if err != nil {
		return a.handleErr(err)
	}
	desc, err := a.readDescription("> Enter new description: ")
	if err != nil {
		return a.handleErr(err)
	}
	if err := a.store.Edit(a.ctx, id, desc); err != nil {
		return a.handleErr(err)
	}
	fmt.Println("Todo updated successfully.")
	return nil
}
