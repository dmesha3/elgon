package migrate

import (
	"crypto/rand"
	stdsql "database/sql"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dmesha3/elgon/orm"
)

// GeneratedMigration holds generated migration metadata and SQL.
type GeneratedMigration struct {
	Version  int
	Name     string
	UpPath   string
	DownPath string
	UpSQL    string
	DownSQL  string
}

type modelEntity struct {
	Table string
	Value any
}

// GenerateFromModelFiles parses model structs from Go files and writes migration SQL files.
//
// Exported struct types become tables. Fields and column options are inferred from `orm` tags
// using the same rules as orm.BuildCreateTableSQL.
func GenerateFromModelFiles(dir, dialect, name string, modelFiles []string) (GeneratedMigration, error) {
	if len(modelFiles) == 0 {
		return GeneratedMigration{}, fmt.Errorf("migrate: no model files provided")
	}
	if strings.TrimSpace(dir) == "" {
		dir = "migrations"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return GeneratedMigration{}, err
	}

	entities, err := parseModelEntities(modelFiles)
	if err != nil {
		return GeneratedMigration{}, err
	}
	if len(entities) == 0 {
		return GeneratedMigration{}, fmt.Errorf("migrate: no exported model structs found")
	}

	upLines := make([]string, 0, len(entities))
	downLines := make([]string, 0, len(entities))
	for _, entity := range entities {
		stmt, err := orm.BuildCreateTableSQLForTable(entity.Table, entity.Value, dialect)
		if err != nil {
			return GeneratedMigration{}, err
		}
		upLines = append(upLines, stmt+";")
	}
	for i := len(entities) - 1; i >= 0; i-- {
		downLines = append(downLines, fmt.Sprintf("DROP TABLE IF EXISTS %s;", entities[i].Table))
	}

	upSQL := strings.Join(upLines, "\n") + "\n"
	downSQL := strings.Join(downLines, "\n") + "\n"

	ver, err := strconv.Atoi(time.Now().UTC().Format("20060102150405"))
	if err != nil {
		return GeneratedMigration{}, err
	}
	base := normalizeName(name)

	var upPath, downPath, finalName string
	for i := 0; i < 16; i++ {
		sfx, err := randomToken(3)
		if err != nil {
			return GeneratedMigration{}, err
		}
		finalName = base + "_" + sfx
		upPath = filepath.Join(dir, fmt.Sprintf("%d_%s.up.sql", ver, finalName))
		downPath = filepath.Join(dir, fmt.Sprintf("%d_%s.down.sql", ver, finalName))
		if _, err := os.Stat(upPath); os.IsNotExist(err) {
			if _, err := os.Stat(downPath); os.IsNotExist(err) {
				break
			}
		}
		if i == 15 {
			return GeneratedMigration{}, fmt.Errorf("migrate: could not allocate unique migration file name")
		}
	}

	if err := os.WriteFile(upPath, []byte(upSQL), 0o644); err != nil {
		return GeneratedMigration{}, err
	}
	if err := os.WriteFile(downPath, []byte(downSQL), 0o644); err != nil {
		_ = os.Remove(upPath)
		return GeneratedMigration{}, err
	}

	return GeneratedMigration{
		Version:  ver,
		Name:     finalName,
		UpPath:   upPath,
		DownPath: downPath,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
	}, nil
}

func parseModelEntities(files []string) ([]modelEntity, error) {
	out := make([]modelEntity, 0)
	for _, file := range files {
		entities, err := parseModelEntitiesFromFile(file)
		if err != nil {
			return nil, err
		}
		out = append(out, entities...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Table < out[j].Table
	})
	return out, nil
}

func parseModelEntitiesFromFile(path string) ([]modelEntity, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	imports := map[string]string{}
	for _, imp := range node.Imports {
		pkgPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}
		name := ""
		if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
			name = imp.Name.Name
		}
		if name == "" {
			parts := strings.Split(pkgPath, "/")
			name = parts[len(parts)-1]
		}
		imports[name] = pkgPath
	}

	entities := make([]modelEntity, 0)
	for _, decl := range node.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !ast.IsExported(ts.Name.Name) {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			fields, err := buildReflectFields(st, imports)
			if err != nil {
				return nil, fmt.Errorf("migrate: %s: type %s: %w", path, ts.Name.Name, err)
			}
			if len(fields) == 0 {
				continue
			}

			rt := reflect.StructOf(fields)
			entities = append(entities, modelEntity{
				Table: toSnakeCase(ts.Name.Name),
				Value: reflect.New(rt).Elem().Interface(),
			})
		}
	}

	sort.SliceStable(entities, func(i, j int) bool {
		return entities[i].Table < entities[j].Table
	})
	return entities, nil
}

func buildReflectFields(st *ast.StructType, imports map[string]string) ([]reflect.StructField, error) {
	if st.Fields == nil {
		return nil, nil
	}
	out := make([]reflect.StructField, 0, len(st.Fields.List))
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		rtype, err := astExprToReflectType(field.Type, imports)
		if err != nil {
			return nil, err
		}

		var tag reflect.StructTag
		if field.Tag != nil {
			raw, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				return nil, err
			}
			tag = reflect.StructTag(raw)
		}

		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			out = append(out, reflect.StructField{
				Name: name.Name,
				Type: rtype,
				Tag:  tag,
			})
		}
	}
	return out, nil
}

func astExprToReflectType(expr ast.Expr, imports map[string]string) (reflect.Type, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return identType(t.Name)
	case *ast.StarExpr:
		inner, err := astExprToReflectType(t.X, imports)
		if err != nil {
			return nil, err
		}
		return reflect.PointerTo(inner), nil
	case *ast.ArrayType:
		if t.Len != nil {
			return nil, fmt.Errorf("fixed-size arrays are not supported")
		}
		elem, err := astExprToReflectType(t.Elt, imports)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(elem), nil
	case *ast.SelectorExpr:
		pkgIdent, ok := t.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("unsupported selector type")
		}
		pkgPath := imports[pkgIdent.Name]
		return selectorType(pkgPath, t.Sel.Name)
	default:
		return nil, fmt.Errorf("unsupported type expression %T", expr)
	}
}

func identType(name string) (reflect.Type, error) {
	switch name {
	case "bool":
		return reflect.TypeOf(false), nil
	case "string":
		return reflect.TypeOf(""), nil
	case "int":
		return reflect.TypeOf(int(0)), nil
	case "int8":
		return reflect.TypeOf(int8(0)), nil
	case "int16":
		return reflect.TypeOf(int16(0)), nil
	case "int32":
		return reflect.TypeOf(int32(0)), nil
	case "int64":
		return reflect.TypeOf(int64(0)), nil
	case "uint":
		return reflect.TypeOf(uint(0)), nil
	case "uint8", "byte":
		return reflect.TypeOf(uint8(0)), nil
	case "uint16":
		return reflect.TypeOf(uint16(0)), nil
	case "uint32":
		return reflect.TypeOf(uint32(0)), nil
	case "uint64":
		return reflect.TypeOf(uint64(0)), nil
	case "float32":
		return reflect.TypeOf(float32(0)), nil
	case "float64":
		return reflect.TypeOf(float64(0)), nil
	default:
		return nil, fmt.Errorf("unsupported identifier type %q", name)
	}
}

func selectorType(pkgPath, sel string) (reflect.Type, error) {
	switch pkgPath {
	case "time":
		if sel == "Time" {
			return reflect.TypeOf(time.Time{}), nil
		}
	case "database/sql":
		switch sel {
		case "NullString":
			return reflect.TypeOf(stdsql.NullString{}), nil
		case "NullBool":
			return reflect.TypeOf(stdsql.NullBool{}), nil
		case "NullInt16":
			return reflect.TypeOf(stdsql.NullInt16{}), nil
		case "NullInt32":
			return reflect.TypeOf(stdsql.NullInt32{}), nil
		case "NullInt64":
			return reflect.TypeOf(stdsql.NullInt64{}), nil
		case "NullByte":
			return reflect.TypeOf(stdsql.NullByte{}), nil
		case "NullFloat64":
			return reflect.TypeOf(stdsql.NullFloat64{}), nil
		case "NullTime":
			return reflect.TypeOf(stdsql.NullTime{}), nil
		}
	}
	return nil, fmt.Errorf("unsupported selector type %s.%s", pkgPath, sel)
}

func normalizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "autogen"
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "autogen"
	}
	return out
}

func randomToken(bytesN int) (string, error) {
	if bytesN <= 0 {
		bytesN = 3
	}
	b := make([]byte, bytesN)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes) + 4)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && ((runes[i-1] >= 'a' && runes[i-1] <= 'z') || (i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z')) {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
