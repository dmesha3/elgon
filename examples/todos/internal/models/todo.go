package models

type Todo struct {
	// elgon.BaseModel `elgon:"table:todos,alias:t"`
	ID          string `elgon:"primary_key"`
	Title       string `elgon:"not_null"`
	Description string `elgon:"not_null"`
	IsCompleted bool   `elgon:"bool"`
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