package todo

type TodoStorage interface {
	Load() ([]Todo, error)
	Save(todos []Todo) error
}
