package services

import (
	"context"

	"github.com/dmesha3/todos/internal/models"
	"github.com/dmesha3/todos/internal/repositories"
)

type TodoService interface {
	GetAllTodos(ctx context.Context) ([]models.Todo, error)
	CreateTodo(ctx context.Context, request models.CreateTodoRequest) (*models.Todo, error)
}

type todoService struct {
	repo repositories.TodoRepository
}

func NewTodoService(repo repositories.TodoRepository) TodoService {
	return &todoService{repo: repo}
}

func (s *todoService) GetAllTodos(ctx context.Context) ([]models.Todo, error) {
	return s.repo.FindAll(ctx)
}

func (s *todoService) CreateTodo(ctx context.Context, request models.CreateTodoRequest) (*models.Todo, error) {
	return s.repo.Create(ctx, request)
}