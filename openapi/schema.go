package openapi

import (
	"reflect"
	"strings"
	"time"
)

func buildSchema(t reflect.Type, components map[string]any) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}

	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": buildSchema(t.Elem(), components)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": buildSchema(t.Elem(), components)}
	case reflect.Struct:
		properties := map[string]any{}
		required := make([]string, 0)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			name, omit := jsonFieldName(f)
			if name == "" {
				continue
			}
			ft := f.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft.Name() != "" && ft.Name() != t.Name() {
				if _, ok := components[ft.Name()]; !ok {
					components[ft.Name()] = buildSchema(ft, components)
				}
				properties[name] = map[string]any{"$ref": "#/components/schemas/" + ft.Name()}
			} else {
				properties[name] = buildSchema(f.Type, components)
			}
			if !omit {
				required = append(required, name)
			}
		}
		schema := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	default:
		return map[string]any{"type": "string"}
	}
}

func jsonFieldName(f reflect.StructField) (name string, omitempty bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if tag == "" {
		return f.Name, false
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		name = f.Name
	} else {
		name = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}
