package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/meshackkazimoto/elgon/db"
	"github.com/meshackkazimoto/elgon/examples/prod-api/internal/domain"
)

type TodoRepo struct {
	DB      db.Adapter
	Dialect string
}

func (r *TodoRepo) Create(ctx context.Context, title string) (domain.Todo, error) {
	now := time.Now().UTC()
	
	if r.isPostgres() {
		rows, err := r.DB.QueryContext(ctx,
			fmt.Sprintf("INSERT INTO todos (title, done, created_at) VALUES (%s, false, %s) RETURNING id, done, created_at", r.ph(1), r.ph(2)),
			title, now,
		)
		if err != nil {
			return domain.Todo{}, err
		}
		defer rows.Close()
		if !rows.Next() {
			return domain.Todo{}, fmt.Errorf("insert todo: no row returned")
		}
		var id int64
		var done bool
		var created time.Time
		if err := rows.Scan(&id, &done, &created); err != nil {
			return domain.Todo{}, err
		}
		return domain.Todo{ID: id, Title: title, Done: done, CreatedAt: created.UTC()}, rows.Err()
	}
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
		var doneRaw any
		var createdRaw any
		if err := rows.Scan(&t.ID, &t.Title, &doneRaw, &createdRaw); err != nil {
			return nil, err
		}
		done, err := toBool(doneRaw)
		if err != nil {
			return nil, err
		}
		created, err := toTime(createdRaw)
		if err != nil {
			return nil, err
		}
		t.Done = done
		t.CreatedAt = created
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TodoRepo) MarkDone(ctx context.Context, id int64) (domain.Todo, bool, error) {
	res, err := r.DB.ExecContext(ctx, fmt.Sprintf("UPDATE todos SET done=1 WHERE id=%s", r.ph(1)), id)
	if err != nil {
		return domain.Todo{}, false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.Todo{}, false, nil
	}
	rows, err := r.DB.QueryContext(ctx, fmt.Sprintf("SELECT id, title, done, created_at FROM todos WHERE id=%s", r.ph(1)), id)
	if err != nil {
		return domain.Todo{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return domain.Todo{}, false, nil
	}
	var t domain.Todo
	var doneRaw any
	var createdRaw any
	if err := rows.Scan(&t.ID, &t.Title, &doneRaw, &createdRaw); err != nil {
		return domain.Todo{}, false, err
	}
	t.Done, err = toBool(doneRaw)
	if err != nil {
		return domain.Todo{}, false, err
	}
	t.CreatedAt, err = toTime(createdRaw)
	if err != nil {
		return domain.Todo{}, false, err
	}
	return t, true, nil
}

func (r *TodoRepo) CreateUser(ctx context.Context, email, name string) (domain.User, error) {
	now := time.Now().UTC()
	if r.isPostgres() {
		rows, err := r.DB.QueryContext(
			ctx,
			fmt.Sprintf("INSERT INTO users (email, name, created_at) VALUES (%s, %s, %s) RETURNING id, created_at", r.ph(1), r.ph(2), r.ph(3)),
			email, name, now,
		)
		if err != nil {
			return domain.User{}, err
		}
		defer rows.Close()
		if !rows.Next() {
			return domain.User{}, fmt.Errorf("insert user: no row returned")
		}
		var id int64
		var created time.Time
		if err := rows.Scan(&id, &created); err != nil {
			return domain.User{}, err
		}
		return domain.User{ID: id, Email: email, Name: name, CreatedAt: created.UTC()}, rows.Err()
	}

	res, err := r.DB.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO users (email, name, created_at) VALUES (%s, %s, %s)", r.ph(1), r.ph(2), r.ph(3)),
		email, name, now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return domain.User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: id, Email: email, Name: name, CreatedAt: now}, nil
}

func (r *TodoRepo) isPostgres() bool {
	return strings.EqualFold(r.Dialect, "postgres") || strings.EqualFold(r.Dialect, "pg")
}

func (r *TodoRepo) ph(n int) string {
	if r.isPostgres() {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func toBool(v any) (bool, error) {
	switch t := v.(type) {
	case bool:
		return t, nil
	case int64:
		return t != 0, nil
	case int32:
		return t != 0, nil
	case int:
		return t != 0, nil
	case []byte:
		return toBool(string(t))
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "1" || s == "t" || s == "true", nil
	default:
		return false, fmt.Errorf("unsupported bool value type %T", v)
	}
}

func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t.UTC(), nil
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, t)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse created_at: %w", err)
		}
		return parsed.UTC(), nil
	case []byte:
		return toTime(string(t))
	default:
		return time.Time{}, fmt.Errorf("unsupported time value type %T", v)
	}
}
