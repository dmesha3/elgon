package migrate

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/meshackkazimoto/elgon/db"
)

const DefaultTable = "schema_migrations"

// Migration is a single versioned SQL migration pair.
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// Status represents migration state in DB.
type Status struct {
	Version int
	Name    string
	Applied bool
}

// Engine runs migration operations using a DB adapter.
type Engine struct {
	DB      db.Adapter
	Table   string
	Dialect string
}

func NewEngine(adapter db.Adapter, dialect string) *Engine {
	return &Engine{DB: adapter, Table: DefaultTable, Dialect: dialect}
}

func (e *Engine) ensureTable(ctx context.Context) error {
	table := e.tableName()
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMP NOT NULL)", table)
	_, err := e.DB.ExecContext(ctx, query)
	return err
}

func (e *Engine) Up(ctx context.Context, migrations []Migration, steps int) (int, error) {
	if err := e.ensureTable(ctx); err != nil {
		return 0, err
	}
	applied, err := e.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	count := 0
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if steps > 0 && count >= steps {
			break
		}
		if strings.TrimSpace(m.UpSQL) == "" {
			return count, fmt.Errorf("migrate: missing up sql for version %d", m.Version)
		}
		if err := e.applyOne(ctx, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (e *Engine) Down(ctx context.Context, migrations []Migration, steps int) (int, error) {
	if err := e.ensureTable(ctx); err != nil {
		return 0, err
	}
	applied, err := e.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version > migrations[j].Version })
	count := 0
	for _, m := range migrations {
		if !applied[m.Version] {
			continue
		}
		if steps > 0 && count >= steps {
			break
		}
		if strings.TrimSpace(m.DownSQL) == "" {
			return count, fmt.Errorf("migrate: missing down sql for version %d", m.Version)
		}
		if err := e.rollbackOne(ctx, m); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (e *Engine) Status(ctx context.Context, migrations []Migration) ([]Status, error) {
	if err := e.ensureTable(ctx); err != nil {
		return nil, err
	}
	applied, err := e.appliedVersions(ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	out := make([]Status, 0, len(migrations))
	for _, m := range migrations {
		out = append(out, Status{Version: m.Version, Name: m.Name, Applied: applied[m.Version]})
	}
	return out, nil
}

func (e *Engine) applyOne(ctx context.Context, m Migration) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
		_ = tx.Rollback()
		return err
	}
	query := fmt.Sprintf("INSERT INTO %s (version, name, applied_at) VALUES (?, ?, CURRENT_TIMESTAMP)", e.tableName())
	if _, err := tx.ExecContext(ctx, query, m.Version, m.Name); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (e *Engine) rollbackOne(ctx context.Context, m Migration) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, m.DownSQL); err != nil {
		_ = tx.Rollback()
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE version = ?", e.tableName())
	if _, err := tx.ExecContext(ctx, query, m.Version); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (e *Engine) appliedVersions(ctx context.Context) (map[int]bool, error) {
	query := fmt.Sprintf("SELECT version FROM %s", e.tableName())
	rows, err := e.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		out[version] = true
	}
	return out, rows.Err()
}

func (e *Engine) tableName() string {
	if e.Table == "" {
		return DefaultTable
	}
	return e.Table
}

func ParseMigrationFileName(filename, dialect string) (version int, name string, direction string, ok bool) {
	base := filepath.Base(filename)
	if !strings.HasSuffix(base, ".sql") {
		return 0, "", "", false
	}
	trim := strings.TrimSuffix(base, ".sql")
	parts := strings.Split(trim, ".")
	if len(parts) < 2 {
		return 0, "", "", false
	}
	left := parts[0]
	direction = parts[len(parts)-1]
	if direction != "up" && direction != "down" {
		return 0, "", "", false
	}

	if len(parts) == 3 {
		if dialect != "" && parts[1] != dialect {
			return 0, "", "", false
		}
	} else if len(parts) != 2 {
		return 0, "", "", false
	}

	idx := strings.Index(left, "_")
	if idx < 1 || idx == len(left)-1 {
		return 0, "", "", false
	}
	v, err := strconv.Atoi(left[:idx])
	if err != nil {
		return 0, "", "", false
	}
	return v, left[idx+1:], direction, true
}
