package todo

import (
	"fmt"
	"sync"
)

type TodoHandler interface {
	Add(description string)
	List() []Todo
	Delete(id int) error
	SetCompleted(id int) error
	SetIncomplete(id int) error
	Edit(id int, description string) error
	Save() error
}

type TodoStore struct {
	mu      sync.Mutex
	todos   []Todo
	storage TodoStorage
	maxID   int
}

func NewTodoStore(storage TodoStorage) (*TodoStore, error) {
	todos, err := storage.Load()
	if err != nil {
		return nil, err
	}

	maxID := 0
	for _, t := range todos {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	return &TodoStore{
		todos:   todos,
		storage: storage,
		maxID:   maxID,
	}, nil
}

func (s *TodoStore) nextID() int {
	s.maxID++
	return s.maxID
}

func (s *TodoStore) findByID(id int) (int, error) {
	if id <= 0 {
		return -1, fmt.Errorf("invalid id: %d", id)
	}
	for i := range s.todos {
		if s.todos[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("todo with id %d not found", id)
}

func (s *TodoStore) Add(description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = append(s.todos, Todo{ID: s.nextID(), Description: description})
}

func (s *TodoStore) List() []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

func (s *TodoStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	s.todos = append(s.todos[:idx], s.todos[idx+1:]...)
	return nil
}

func (s *TodoStore) SetCompleted(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	if s.todos[idx].Completed {
		return fmt.Errorf("todo %d is already completed", id)
	}
	s.todos[idx].Completed = true
	return nil
}

func (s *TodoStore) SetIncomplete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	if !s.todos[idx].Completed {
		return fmt.Errorf("todo %d is already incomplete", id)
	}
	s.todos[idx].Completed = false
	return nil
}

func (s *TodoStore) Edit(id int, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.findByID(id)
	if err != nil {
		return err
	}
	s.todos[idx].Description = description
	return nil
}

func (s *TodoStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storage.Save(s.todos)
}
