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

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (db.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (db.Rows, error)
}

type columnsProvider interface {
	Columns() ([]string, error)
}

// Table is a generic table-level ORM repository.
type Table struct {
	db      db.Adapter
	dialect string
	name    string
}

// Create inserts one row.
func (t *Table) Create(ctx context.Context, values Values) (db.Result, error) {
	return t.createWith(ctx, t.db, values)
}

// CreateMany inserts multiple rows and returns affected rows count.
func (t *Table) CreateMany(ctx context.Context, rows []Values) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil || tx == nil {
		return t.createManyWithoutTx(ctx, rows)
	}

	var count int64
	for _, row := range rows {
		if _, err := t.createWith(ctx, tx, row); err != nil {
			_ = tx.Rollback()
			return count, err
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return count, err
	}
	return count, nil
}

// CreateManyAndReturn inserts multiple rows and returns inserted records.
func (t *Table) CreateManyAndReturn(ctx context.Context, rows []Values, columns []string) ([]Record, error) {
	if len(rows) == 0 {
		return []Record{}, nil
	}
	if !t.supportsReturning() {
		if _, err := t.CreateMany(ctx, rows); err != nil {
			return nil, err
		}
		return recordsFromValues(rows, columns)
	}

	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil || tx == nil {
		return t.createManyAndReturnWith(ctx, t.db, rows, columns)
	}

	created, err := t.createManyAndReturnWith(ctx, tx, rows, columns)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// FindMany returns all matching records.
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
	return scanRecords(rows, opts.Columns)
}

// FindFirst returns the first matching record, or nil if no row exists.
func (t *Table) FindFirst(ctx context.Context, opts FindOptions) (Record, error) {
	opts.Limit = 1
	records, err := t.FindMany(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

// FindFirstOrThrow returns the first matching record or ErrNotFound.
func (t *Table) FindFirstOrThrow(ctx context.Context, opts FindOptions) (Record, error) {
	record, err := t.FindFirst(ctx, opts)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	return record, nil
}

// FindUnique returns one matching record or nil if no row exists.
// It returns ErrNonUnique when multiple rows match.
func (t *Table) FindUnique(ctx context.Context, where Where, columns ...string) (Record, error) {
	if len(where) == 0 {
		return nil, fmt.Errorf("%w: where is required for unique lookup", ErrInvalidInput)
	}
	opts := FindOptions{
		Columns: columns,
		Where:   where,
		Limit:   2,
	}
	records, err := t.FindMany(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	if len(records) > 1 {
		return nil, ErrNonUnique
	}
	return records[0], nil
}

// FindUniqueOrThrow returns one unique record or ErrNotFound/ErrNonUnique.
func (t *Table) FindUniqueOrThrow(ctx context.Context, where Where, columns ...string) (Record, error) {
	record, err := t.FindUnique(ctx, where, columns...)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	return record, nil
}

// Update updates rows matching where and returns affected count.
func (t *Table) Update(ctx context.Context, where Where, patch Values) (int64, error) {
	return t.updateManyWith(ctx, t.db, where, patch)
}

// UpdateMany updates rows matching where and returns affected count.
func (t *Table) UpdateMany(ctx context.Context, where Where, patch Values) (int64, error) {
	return t.Update(ctx, where, patch)
}

// UpdateManyAndReturn updates rows and returns updated records.
func (t *Table) UpdateManyAndReturn(ctx context.Context, where Where, patch Values, columns []string) ([]Record, error) {
	if len(where) == 0 {
		return nil, fmt.Errorf("%w: where is required for update", ErrInvalidInput)
	}
	if !t.supportsReturning() {
		loadColumns := columns
		if len(loadColumns) == 0 || hasStar(loadColumns) {
			loadColumns = columnHints(where, patch)
		}
		before, err := t.FindMany(ctx, FindOptions{Where: where, Columns: loadColumns})
		if err != nil {
			return nil, err
		}
		if _, err := t.UpdateMany(ctx, where, patch); err != nil {
			return nil, err
		}
		updated := applyPatch(before, patch)
		return projectRecords(updated, columns)
	}

	query, args, err := t.updateQuery(where, patch, columns, true)
	if err != nil {
		return nil, err
	}
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows, columns)
}

// Delete deletes rows matching where and returns affected count.
func (t *Table) Delete(ctx context.Context, where Where) (int64, error) {
	if len(where) == 0 {
		return 0, fmt.Errorf("%w: where is required for delete", ErrInvalidInput)
	}
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return 0, err
	}
	whereSQL, whereArgs, err := t.whereClause(where, 0)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("DELETE FROM %s%s", table, whereSQL)
	res, err := t.db.ExecContext(ctx, query, whereArgs...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteMany deletes rows matching where and returns affected count.
func (t *Table) DeleteMany(ctx context.Context, where Where) (int64, error) {
	return t.Delete(ctx, where)
}

// Upsert updates matching rows when present, otherwise inserts a new row.
func (t *Table) Upsert(ctx context.Context, where Where, create Values, update Values) (Record, error) {
	if len(where) == 0 {
		return nil, fmt.Errorf("%w: where is required for upsert", ErrInvalidInput)
	}
	existing, err := t.FindUnique(ctx, where)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		merged := mergeCreateAndWhere(create, where)
		if _, err := t.Create(ctx, merged); err != nil {
			return nil, err
		}
		return t.FindUniqueOrThrow(ctx, where)
	}
	if len(update) > 0 {
		if _, err := t.UpdateMany(ctx, where, update); err != nil {
			return nil, err
		}
	}
	return t.FindUniqueOrThrow(ctx, where)
}

func (t *Table) createManyWithoutTx(ctx context.Context, rows []Values) (int64, error) {
	var count int64
	for _, row := range rows {
		if _, err := t.Create(ctx, row); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (t *Table) createManyAndReturnWith(ctx context.Context, eq execQuerier, rows []Values, columns []string) ([]Record, error) {
	out := make([]Record, 0, len(rows))
	for _, values := range rows {
		rec, err := t.createAndReturnWith(ctx, eq, values, columns)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (t *Table) createAndReturnWith(ctx context.Context, eq execQuerier, values Values, columns []string) (Record, error) {
	query, args, err := t.insertQuery(values, columns, true)
	if err != nil {
		return nil, err
	}
	rows, err := eq.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err := scanRecords(rows, columns)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, ErrNotFound
	}
	return records[0], nil
}

func (t *Table) createWith(ctx context.Context, eq execQuerier, values Values) (db.Result, error) {
	query, args, err := t.insertQuery(values, nil, false)
	if err != nil {
		return nil, err
	}
	return eq.ExecContext(ctx, query, args...)
}

func (t *Table) insertQuery(values Values, returning []string, withReturning bool) (string, []any, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return "", nil, err
	}
	cols, args, err := normalizedValues(values, "values")
	if err != nil {
		return "", nil, err
	}

	holders := make([]string, len(cols))
	for i := range cols {
		holders[i] = t.ph(i + 1)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(holders, ", "))
	if withReturning {
		clause, err := buildReturningClause(returning)
		if err != nil {
			return "", nil, err
		}
		query += clause
	}
	return query, args, nil
}

func (t *Table) updateManyWith(ctx context.Context, eq execQuerier, where Where, patch Values) (int64, error) {
	query, args, err := t.updateQuery(where, patch, nil, false)
	if err != nil {
		return 0, err
	}
	res, err := eq.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t *Table) updateQuery(where Where, patch Values, returning []string, withReturning bool) (string, []any, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return "", nil, err
	}
	if len(where) == 0 {
		return "", nil, fmt.Errorf("%w: where is required for update", ErrInvalidInput)
	}

	setCols, setArgs, err := normalizedValues(patch, "patch")
	if err != nil {
		return "", nil, err
	}
	whereSQL, whereArgs, err := t.whereClause(where, len(setArgs))
	if err != nil {
		return "", nil, err
	}

	setParts := make([]string, 0, len(setCols))
	for i, col := range setCols {
		setParts = append(setParts, fmt.Sprintf("%s = %s", col, t.ph(i+1)))
	}
	args := make([]any, 0, len(setArgs)+len(whereArgs))
	args = append(args, setArgs...)
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s%s", table, strings.Join(setParts, ", "), whereSQL)
	if withReturning {
		clause, err := buildReturningClause(returning)
		if err != nil {
			return "", nil, err
		}
		query += clause
	}
	return query, args, nil
}

func (t *Table) selectQuery(opts FindOptions) (string, []any, error) {
	table, err := validIdentifier(t.name, "table")
	if err != nil {
		return "", nil, err
	}

	columns, err := selectColumns(opts.Columns)
	if err != nil {
		return "", nil, err
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

func (t *Table) supportsReturning() bool {
	dialect := strings.ToLower(strings.TrimSpace(t.dialect))
	return dialect == "postgres" || dialect == "pg" || dialect == "sqlite" || dialect == "sqlite3"
}

func (t *Table) ph(n int) string {
	if strings.EqualFold(t.dialect, "postgres") || strings.EqualFold(t.dialect, "pg") {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func scanRecords(rows db.Rows, columnsHint []string) ([]Record, error) {
	cols := columnsHint
	if provider, ok := rows.(columnsProvider); ok {
		var err error
		cols, err = provider.Columns()
		if err != nil {
			return nil, err
		}
	} else if len(cols) == 0 || hasStar(cols) {
		return nil, fmt.Errorf("orm: db rows type does not expose columns")
	}

	out := make([]Record, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		record := make(Record, len(cols))
		for i := range cols {
			record[cols[i]] = normalizeDBValue(values[i])
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func selectColumns(columns []string) (string, error) {
	if len(columns) == 0 {
		return "*", nil
	}
	validCols := make([]string, 0, len(columns))
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col == "*" {
			return "*", nil
		}
		v, err := validIdentifier(col, "column")
		if err != nil {
			return "", err
		}
		validCols = append(validCols, v)
	}
	return strings.Join(validCols, ", "), nil
}

func buildReturningClause(columns []string) (string, error) {
	cols, err := selectColumns(columns)
	if err != nil {
		return "", err
	}
	return " RETURNING " + cols, nil
}

func recordsFromValues(rows []Values, columns []string) ([]Record, error) {
	out := make([]Record, 0, len(rows))
	for _, row := range rows {
		record := make(Record, len(row))
		for k, v := range row {
			record[k] = v
		}
		out = append(out, record)
	}
	return projectRecords(out, columns)
}

func projectRecords(rows []Record, columns []string) ([]Record, error) {
	if len(columns) == 0 || hasStar(columns) {
		return rows, nil
	}
	projected := make([]Record, 0, len(rows))
	for _, row := range rows {
		next := make(Record, len(columns))
		for _, col := range columns {
			validCol, err := validIdentifier(col, "column")
			if err != nil {
				return nil, err
			}
			if v, ok := row[validCol]; ok {
				next[validCol] = v
			}
		}
		projected = append(projected, next)
	}
	return projected, nil
}

func applyPatch(rows []Record, patch Values) []Record {
	out := make([]Record, 0, len(rows))
	for _, row := range rows {
		next := make(Record, len(row)+len(patch))
		for k, v := range row {
			next[k] = v
		}
		for k, v := range patch {
			next[k] = v
		}
		out = append(out, next)
	}
	return out
}

func mergeCreateAndWhere(create Values, where Where) Values {
	merged := make(Values, len(create)+len(where))
	for k, v := range where {
		merged[k] = v
	}
	for k, v := range create {
		merged[k] = v
	}
	return merged
}

func columnHints(where Where, patch Values) []string {
	hints := make([]string, 0, len(where)+len(patch))
	seen := map[string]struct{}{}
	for k := range where {
		hints = append(hints, k)
		seen[k] = struct{}{}
	}
	for k := range patch {
		if _, ok := seen[k]; ok {
			continue
		}
		hints = append(hints, k)
	}
	sort.Strings(hints)
	return hints
}

func hasStar(columns []string) bool {
	for _, col := range columns {
		if strings.TrimSpace(col) == "*" {
			return true
		}
	}
	return false
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
