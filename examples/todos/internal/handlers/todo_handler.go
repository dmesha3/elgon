package handlers

import (
	"net/http"

	"github.com/dmesha3/elgon"
	"github.com/dmesha3/elgon/openapi"
	"github.com/dmesha3/todos/internal/models"
	"github.com/dmesha3/todos/internal/services"
)

type TodoHandler struct {
	service services.TodoService
	openapi   *openapi.Generator
}

func NewTodoHandler(service services.TodoService, docs *openapi.Generator) *TodoHandler {
	return &TodoHandler{service: service, openapi: docs,}
}

func (h *TodoHandler) RegisterDocs() {
	h.openapi.AddOperation("GET", "/api/v1/todos", openapi.Operation{
		Summary:       "Get all todos",
		Description:   "Fetch all todo items",
		OperationID:   "getAllTodos",
		Tags:          []string{"Todos"},
		ResponseModel: []models.TodoResponse{},
		ResponseCode:  http.StatusOK,
	})

	h.openapi.AddOperation("POST", "/api/v1/todos", openapi.Operation{
		Summary:       "Create a todo",
		Description:   "Create a new todo item",
		OperationID:   "createTodo",
		Tags:          []string{"Todos"},
		RequestModel:  models.CreateTodoRequest{},
		ResponseModel: models.TodoResponse{},
		ResponseCode:  http.StatusCreated,
		RequestExample: map[string]any{
			"title": "Buy milk",
			"isComplete":  false,
		},
		ResponseExample: map[string]any{
			"id":    "1",
			"title": "Buy milk",
			"done":  false,
		},
	})
}

func (h *TodoHandler) GetAll(c *elgon.Ctx) error {
	ctx := c.Request.Context()

	todos, err := h.service.GetAllTodos(ctx)
	if err != nil {
		return elgon.ErrInternal("Failed to fetch todos")
	}

	return c.JSON(http.StatusOK, todos)
}

func (h *TodoHandler) CreateTodo(c *elgon.Ctx) error {
	ctx := c.Request.Context()

	var req models.CreateTodoRequest

	if err := c.BindJSON(&req); err != nil {
		return elgon.ErrBadRequest("Invalid request body", "")
	}

	if req.Title == "" {
		return elgon.ErrBadRequest("Title is required", "")
	}

	todo, err := h.service.CreateTodo(ctx, req)
	if err != nil {
		return elgon.ErrInternal("Failed to create todo")
	}

	return c.JSON(http.StatusCreated, todo)
}