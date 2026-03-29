package routes

import (
	"github.com/dmesha3/elgon"
	"github.com/dmesha3/todos/internal/handlers"
)

func RegisterTodoRoutes(app *elgon.App, handler *handlers.TodoHandler) {
	api := app.Group("/api/v1")

	api.GET("/todos", handler.GetAll)
	api.POST("/todos", handler.CreateTodo)

	handler.RegisterDocs()
}