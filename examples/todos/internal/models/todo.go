package models

type Todo struct {
	ID          string ``
	Title       string ``
	Description string ``
	IsCompleted bool   ``
}

type CreateTodoRequest struct {
	Title       string
	Description string
}

type TodoResponse struct {
	ID          string ``
	Title       string ``
	Description string ``
	IsCompleted bool   ``
}