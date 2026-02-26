package orm

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/meshackkazimoto/elgon/db"
)

type Values map[string]any
type Where map[string]any
type Record map[string]any

type OrderBy struct {
	Column string
	Desc   bool
}

type FindOptions struct {
	Columns []string
	Where   Where
	OrderBy []OrderBy
	Limit   int
	Offset  int
}

// Table is a generic table-level ORM repository.
type Table struct {
	db      db.Adapter
	dialect string
	name    string
}

// Create inserts a single record and returns the raw database result.
func (t *Table) Create(ctx context.Context, values Values) (db.Result, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return nil, err
	}
	cols, args, err := normalizedValues(values, "values")
	if err != nil {
		return nil, err
	}

	holders := make([]string, len(cols))
	for i := range cols {
		holders[i] = t.ph(i + 1)
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(cols, ", "),
		strings.Join(holders, ", "),
	)
	return t.db.ExecContext(ctx, query, args...)
}

// FindOne loads one record matching the filter.
func (t *Table) FindOne(ctx context.Context, opts FindOptions) (Record, error) {
	opts.Limit = 1
	rows, err := t.FindMany(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

// FindMany loads many records matching the filter.
func (t *Table) FindMany(ctx context.Context, opts FindOptions) ([]Record, error) {
	query, args, err := t.selectQuery(opts)
	if err != nil {
		return nil, err
	}

	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	provider, ok := rows.(interface{ Columns() ([]string, error) })
	if !ok {
		return nil, fmt.Errorf("orm: db rows type does not expose columns")
	}
	columns, err := provider.Columns()
	if err != nil {
		return nil, err
	}

	out := make([]Record, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(Record, len(columns))
		for i := range columns {
			row[columns[i]] = normalizeDBValue(values[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// Update applies a partial update. Empty where is rejected to avoid full-table updates.
func (t *Table) Update(ctx context.Context, where Where, patch Values) (int64, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return 0, err
	}
	setCols, setArgs, err := normalizedValues(patch, "patch")
	if err != nil {
		return 0, err
	}
	whereSQL, whereArgs, err := t.whereClause(where, len(setArgs))
	if err != nil {
		return 0, err
	}
	if whereSQL == "" {
		return 0, fmt.Errorf("%w: where is required for update", ErrInvalidInput)
	}

	setParts := make([]string, 0, len(setCols))
	for i, col := range setCols {
		setParts = append(setParts, fmt.Sprintf("%s = %s", col, t.ph(i+1)))
	}

	args := make([]any, 0, len(setArgs)+len(whereArgs))
	args = append(args, setArgs...)
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s%s", table, strings.Join(setParts, ", "), whereSQL)
	res, err := t.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// Delete removes rows matching where. Empty where is rejected to avoid full-table deletes.
func (t *Table) Delete(ctx context.Context, where Where) (int64, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return 0, err
	}
	whereSQL, whereArgs, err := t.whereClause(where, 0)
	if err != nil {
		return 0, err
	}
	if whereSQL == "" {
		return 0, fmt.Errorf("%w: where is required for delete", ErrInvalidInput)
	}

	query := fmt.Sprintf("DELETE FROM %s%s", table, whereSQL)
	res, err := t.db.ExecContext(ctx, query, whereArgs...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (t *Table) selectQuery(opts FindOptions) (string, []any, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return "", nil, err
	}

	columns := "*"
	if len(opts.Columns) > 0 {
		validCols := make([]string, 0, len(opts.Columns))
		for _, col := range opts.Columns {
			col = strings.TrimSpace(col)
			if col == "*" {
				validCols = []string{"*"}
				break
			}
			v, err := validIdentifier(col, "column")
			if err != nil {
				return "", nil, err
			}
			validCols = append(validCols, v)
		}
		columns = strings.Join(validCols, ", ")
	}

	whereSQL, whereArgs, err := t.whereClause(opts.Where, 0)
	if err != nil {
		return "", nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM %s%s", columns, table, whereSQL)
	args := whereArgs

	if len(opts.OrderBy) > 0 {
		orderParts := make([]string, 0, len(opts.OrderBy))
		for _, order := range opts.OrderBy {
			col, err := validIdentifier(order.Column, "order column")
			if err != nil {
				return "", nil, err
			}
			if order.Desc {
				orderParts = append(orderParts, col+" DESC")
			} else {
				orderParts = append(orderParts, col+" ASC")
			}
		}
		query += " ORDER BY " + strings.Join(orderParts, ", ")
	}

	next := len(args) + 1
	if opts.Limit > 0 {
		query += " LIMIT " + t.ph(next)
		args = append(args, opts.Limit)
		next++
	}
	if opts.Offset > 0 {
		query += " OFFSET " + t.ph(next)
		args = append(args, opts.Offset)
	}
	return query, args, nil
}

func (t *Table) whereClause(where Where, start int) (string, []any, error) {
	if len(where) == 0 {
		return "", nil, nil
	}
	cols := make([]string, 0, len(where))
	for col := range where {
		cols = append(cols, col)
	}
	sort.Strings(cols)

	parts := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))
	for i, col := range cols {
		validCol, err := validIdentifier(col, "where column")
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("%s = %s", validCol, t.ph(start+i+1)))
		args = append(args, where[col])
	}
	return " WHERE " + strings.Join(parts, " AND "), args, nil
}

func (t *Table) ph(n int) string {
	if strings.EqualFold(t.dialect, "postgres") || strings.EqualFold(t.dialect, "pg") {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func normalizedValues(values Values, label string) ([]string, []any, error) {
	if len(values) == 0 {
		return nil, nil, fmt.Errorf("%w: %s must not be empty", ErrInvalidInput, label)
	}
	cols := make([]string, 0, len(values))
	for col := range values {
		cols = append(cols, col)
	}
	sort.Strings(cols)

	args := make([]any, 0, len(cols))
	validCols := make([]string, 0, len(cols))
	for _, col := range cols {
		validCol, err := validIdentifier(col, "column")
		if err != nil {
			return nil, nil, err
		}
		validCols = append(validCols, validCol)
		args = append(args, values[col])
	}
	return validCols, args, nil
}

var identRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validIdentifier(name, label string) (string, error) {
	name = strings.TrimSpace(name)
	if !identRegex.MatchString(name) {
		return "", fmt.Errorf("%w: invalid %s %q", ErrInvalidInput, label, name)
	}
	return name, nil
}

func normalizeDBValue(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}
