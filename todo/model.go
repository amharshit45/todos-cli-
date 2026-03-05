package todo

type Todo struct {
	ID          int    `json:"id" bson:"id"`
	Description string `json:"description" bson:"description"`
	Completed   bool   `json:"completed" bson:"completed"`
}
