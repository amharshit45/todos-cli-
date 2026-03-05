package todo

type TodoStorage interface {
	Add(description string) error
	List() ([]Todo, error)
	Delete(id int) error
	SetCompleted(id int) error
	SetIncomplete(id int) error
	Edit(id int, description string) error
	Close() error
}
