package repositories

import (
	"context"

	"github.com/dmesha3/elgon/db"
	"github.com/dmesha3/todos/internal/models"
	"github.com/google/uuid"
)

type TodoRepository interface {
	FindAll(ctx context.Context) ([]models.Todo, error)
	Create(ctx context.Context, request models.CreateTodoRequest) (*models.Todo, error)
}

type todoRepository struct {
	db db.Adapter
}

func NewTodoRepository(db db.Adapter) TodoRepository {
	return &todoRepository{
		db: db,
	}
}

func (r *todoRepository) FindAll(ctx context.Context) ([]models.Todo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, description, is_completed 
		FROM todos
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []models.Todo

	for rows.Next() {
		var t models.Todo
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.IsCompleted)
		if err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}

	return todos, nil
}

func (r *todoRepository) Create(ctx context.Context, request models.CreateTodoRequest) (*models.Todo, error) {
	var todos models.Todo

	id := uuid.New().String()

	rows, err := r.db.QueryContext(ctx, `
	INSERT INTO todos (id, title, description, is_completed)
	VALUES ($1, $2, $3, $4)
	RETURNING id, title, description, is_completed
	`,
		id,
		request.Title,
		request.Description,
		false,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var t models.Todo
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.IsCompleted)
		if err != nil {
			return nil, err
		}
		todos = t
	}

	return &todos, nil
}
