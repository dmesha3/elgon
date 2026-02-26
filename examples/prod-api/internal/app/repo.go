package app

import (
	"context"
	"fmt"
	"time"

	"github.com/meshackkazimoto/elgon/db"
	"github.com/meshackkazimoto/elgon/examples/prod-api/internal/domain"
)

type TodoRepo struct {
	DB db.Adapter
}

func (r *TodoRepo) Create(ctx context.Context, title string) (domain.Todo, error) {
	now := time.Now().UTC()
	res, err := r.DB.ExecContext(ctx, "INSERT INTO todos (title, done, created_at) VALUES (?, 0, ?)", title, now.Format(time.RFC3339Nano))
	if err != nil {
		return domain.Todo{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Todo{}, err
	}
	return domain.Todo{ID: id, Title: title, Done: false, CreatedAt: now}, nil
}

func (r *TodoRepo) List(ctx context.Context) ([]domain.Todo, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, title, done, created_at FROM todos ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Todo, 0)
	for rows.Next() {
		var t domain.Todo
		var done int
		var created string
		if err := rows.Scan(&t.ID, &t.Title, &done, &created); err != nil {
			return nil, err
		}
		t.Done = done == 1
		parsed, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		t.CreatedAt = parsed
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TodoRepo) MarkDone(ctx context.Context, id int64) (domain.Todo, bool, error) {
	res, err := r.DB.ExecContext(ctx, "UPDATE todos SET done=1 WHERE id=?", id)
	if err != nil {
		return domain.Todo{}, false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.Todo{}, false, nil
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id, title, done, created_at FROM todos WHERE id=?", id)
	if err != nil {
		return domain.Todo{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return domain.Todo{}, false, nil
	}
	var t domain.Todo
	var done int
	var created string
	if err := rows.Scan(&t.ID, &t.Title, &done, &created); err != nil {
		return domain.Todo{}, false, err
	}
	t.Done = done == 1
	t.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return domain.Todo{}, false, err
	}
	return t, true, nil
}
