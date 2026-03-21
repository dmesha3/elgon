package orm

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/dmesha3/elgon/db"
)

type tableNamer interface {
	TableName() string
}

type execContexter interface {
	ExecContext(ctx context.Context, query string, args ...any) (db.Result, error)
}

type schemaFieldOptions struct {
	skip          bool
	column        string
	sqlType       string
	defaultExpr   string
	size          int
	primaryKey    bool
	autoIncrement bool
	notNull       bool
	unique        bool
}

// AutoMigrate creates missing tables for the provided entities.
// It is additive only and does not alter existing columns.
func (c *Client) AutoMigrate(ctx context.Context, entities ...any) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("%w: orm client is not configured", ErrInvalidInput)
	}
	if len(entities) == 0 {
		return nil
	}

	tx, err := c.db.BeginTx(ctx, nil)
	useTx := err == nil && tx != nil
	execDB := execContexter(c.db)
	if useTx {
		execDB = tx
	}

	for _, entity := range entities {
		stmt, err := BuildCreateTableSQL(entity, c.cfg.Dialect)
		if err != nil {
			if useTx {
				_ = tx.Rollback()
			}
			return err
		}
		if _, err := execDB.ExecContext(ctx, stmt); err != nil {
			if useTx {
				_ = tx.Rollback()
			}
			return err
		}
	}

	if useTx {
		return tx.Commit()
	}
	return nil
}

// BuildCreateTableSQL builds a CREATE TABLE IF NOT EXISTS statement for an entity.
// Entity must be a struct (or pointer to struct). Exported fields become columns.
func BuildCreateTableSQL(entity any, dialect string) (string, error) {
	return buildCreateTableSQL(entity, dialect, "")
}

// BuildCreateTableSQLForTable builds a CREATE TABLE statement for an entity with an explicit table name override.
func BuildCreateTableSQLForTable(table string, entity any, dialect string) (string, error) {
	return buildCreateTableSQL(entity, dialect, table)
}

func buildCreateTableSQL(entity any, dialect, tableOverride string) (string, error) {
	structType, err := entityStructType(entity)
	if err != nil {
		return "", err
	}

	tableName := strings.TrimSpace(tableOverride)
	if tableName == "" {
		tableName, err = entityTableName(entity, structType)
		if err != nil {
			return "", err
		}
	} else {
		tableName, err = validIdentifier(tableName, "table")
		if err != nil {
			return "", err
		}
	}

	cols := make([]string, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" || field.Anonymous {
			continue
		}

		opts, err := parseSchemaFieldTag(field.Tag.Get("orm"))
		if err != nil {
			return "", fmt.Errorf("%w: field %s: %v", ErrInvalidInput, field.Name, err)
		}
		if opts.skip {
			continue
		}

		columnName := opts.column
		if columnName == "" {
			columnName = toSnakeCase(field.Name)
		}
		columnName, err = validIdentifier(columnName, "column")
		if err != nil {
			return "", err
		}

		sqlType, baseType, _, err := inferSQLType(field.Type, opts, dialect)
		if err != nil {
			return "", fmt.Errorf("%w: field %s: %v", ErrInvalidInput, field.Name, err)
		}

		colDef, err := buildColumnDef(columnName, sqlType, baseType, opts, dialect)
		if err != nil {
			return "", fmt.Errorf("%w: field %s: %v", ErrInvalidInput, field.Name, err)
		}
		cols = append(cols, colDef)
	}

	if len(cols) == 0 {
		return "", fmt.Errorf("%w: entity %s has no exported migratable fields", ErrInvalidInput, structType.Name())
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(cols, ", ")), nil
}

func entityStructType(entity any) (reflect.Type, error) {
	if entity == nil {
		return nil, fmt.Errorf("%w: entity must not be nil", ErrInvalidInput)
	}
	t := reflect.TypeOf(entity)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: entity must be a struct or pointer to struct", ErrInvalidInput)
	}
	return t, nil
}

func entityTableName(entity any, structType reflect.Type) (string, error) {
	if entity != nil {
		if namer, ok := entity.(tableNamer); ok {
			name, err := validIdentifier(strings.TrimSpace(namer.TableName()), "table")
			if err != nil {
				return "", err
			}
			return name, nil
		}
	}

	ptr := reflect.New(structType).Interface()
	if namer, ok := ptr.(tableNamer); ok {
		name, err := validIdentifier(strings.TrimSpace(namer.TableName()), "table")
		if err != nil {
			return "", err
		}
		return name, nil
	}

	if structType.Name() == "" {
		return "", fmt.Errorf("%w: cannot infer table name for anonymous struct", ErrInvalidInput)
	}
	name, err := validIdentifier(toSnakeCase(structType.Name()), "table")
	if err != nil {
		return "", err
	}
	return name, nil
}

func parseSchemaFieldTag(raw string) (schemaFieldOptions, error) {
	opts := schemaFieldOptions{size: -1}
	tag := strings.TrimSpace(raw)
	if tag == "" {
		return opts, nil
	}
	if tag == "-" {
		opts.skip = true
		return opts, nil
	}

	for _, token := range strings.Split(tag, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if token == "-" {
			opts.skip = true
			continue
		}

		key, value, hasValue := strings.Cut(token, ":")
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "column":
			if !hasValue || strings.TrimSpace(value) == "" {
				return opts, fmt.Errorf("column requires non-empty value")
			}
			opts.column = strings.TrimSpace(value)
		case "type":
			if !hasValue || strings.TrimSpace(value) == "" {
				return opts, fmt.Errorf("type requires non-empty value")
			}
			opts.sqlType = strings.TrimSpace(value)
		case "default":
			if !hasValue || strings.TrimSpace(value) == "" {
				return opts, fmt.Errorf("default requires non-empty value")
			}
			opts.defaultExpr = strings.TrimSpace(value)
		case "size":
			if !hasValue {
				return opts, fmt.Errorf("size requires value")
			}
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("size must be a positive integer")
			}
			opts.size = n
		case "pk", "primarykey":
			opts.primaryKey = true
		case "autoincrement":
			opts.autoIncrement = true
		case "notnull":
			opts.notNull = true
		case "unique":
			opts.unique = true
		default:
			return opts, fmt.Errorf("unknown orm tag option %q", key)
		}
	}

	return opts, nil
}

func inferSQLType(fieldType reflect.Type, opts schemaFieldOptions, dialect string) (string, reflect.Type, bool, error) {
	baseType, nullable := unwrapType(fieldType)

	if isSQLNullType(baseType) {
		sqlType, err := sqlTypeForSQLNull(baseType)
		return sqlType, baseType, true, err
	}

	if opts.sqlType != "" {
		return opts.sqlType, baseType, nullable, nil
	}

	if baseType == reflect.TypeOf(time.Time{}) {
		return "TIMESTAMP", baseType, nullable, nil
	}

	switch baseType.Kind() {
	case reflect.Bool:
		return "BOOLEAN", baseType, nullable, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER", baseType, nullable, nil
	case reflect.Int64, reflect.Uint64:
		return "BIGINT", baseType, nullable, nil
	case reflect.Float32, reflect.Float64:
		return "REAL", baseType, nullable, nil
	case reflect.String:
		if opts.size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", opts.size), baseType, nullable, nil
		}
		return "TEXT", baseType, nullable, nil
	case reflect.Slice:
		if baseType.Elem().Kind() == reflect.Uint8 {
			return "BLOB", baseType, nullable, nil
		}
	}

	return "", baseType, false, fmt.Errorf("unsupported field type %s", baseType.String())
}

func buildColumnDef(column, sqlType string, baseType reflect.Type, opts schemaFieldOptions, dialect string) (string, error) {
	parts := []string{column, sqlType}
	dialect = normalizeDialect(dialect)

	if opts.autoIncrement {
		if !isIntegerType(baseType) {
			return "", fmt.Errorf("autoincrement requires integer type")
		}
		switch dialect {
		case "sqlite", "sqlite3":
			parts = []string{column, "INTEGER", "PRIMARY KEY", "AUTOINCREMENT"}
			if opts.unique {
				parts = append(parts, "UNIQUE")
			}
			if opts.defaultExpr != "" {
				parts = append(parts, "DEFAULT "+opts.defaultExpr)
			}
			return strings.Join(parts, " "), nil
		case "postgres", "pg":
			parts[1] = "BIGSERIAL"
			opts.primaryKey = true
			opts.notNull = true
		case "mysql":
			opts.primaryKey = true
			opts.notNull = true
		default:
			opts.primaryKey = true
			opts.notNull = true
		}
	}

	if opts.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if opts.notNull || opts.primaryKey {
		parts = append(parts, "NOT NULL")
	}
	if opts.unique && !opts.primaryKey {
		parts = append(parts, "UNIQUE")
	}
	if opts.defaultExpr != "" {
		parts = append(parts, "DEFAULT "+opts.defaultExpr)
	}
	if opts.autoIncrement && dialect == "mysql" {
		parts = append(parts, "AUTO_INCREMENT")
	}

	return strings.Join(parts, " "), nil
}

func unwrapType(t reflect.Type) (reflect.Type, bool) {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	return t, nullable
}

func isIntegerType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func normalizeDialect(dialect string) string {
	return strings.ToLower(strings.TrimSpace(dialect))
}

func isSQLNullType(t reflect.Type) bool {
	return t.PkgPath() == "database/sql" && strings.HasPrefix(t.Name(), "Null")
}

func sqlTypeForSQLNull(t reflect.Type) (string, error) {
	switch t.Name() {
	case "NullString":
		return "TEXT", nil
	case "NullBool":
		return "BOOLEAN", nil
	case "NullInt16", "NullInt32", "NullByte":
		return "INTEGER", nil
	case "NullInt64":
		return "BIGINT", nil
	case "NullFloat64":
		return "REAL", nil
	case "NullTime":
		return "TIMESTAMP", nil
	default:
		return "", fmt.Errorf("unsupported nullable sql type %s.%s", t.PkgPath(), t.Name())
	}
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes) + 4)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
