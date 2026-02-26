package openapi

import (
	"encoding/json"
	"reflect"
	"strconv"
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
			applyFieldAnnotations(properties[name].(map[string]any), f)
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

func applyFieldAnnotations(schema map[string]any, f reflect.StructField) {
	if desc := strings.TrimSpace(f.Tag.Get("description")); desc != "" {
		schema["description"] = desc
	}
	if ex := strings.TrimSpace(f.Tag.Get("example")); ex != "" {
		schema["example"] = typedExample(ex, schema["type"])
	}
	if raw := strings.TrimSpace(f.Tag.Get("openapi")); raw != "" {
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			switch key {
			case "description":
				schema["description"] = val
			case "format":
				schema["format"] = val
			case "example":
				schema["example"] = typedExample(val, schema["type"])
			case "enum":
				items := strings.Split(val, "|")
				enum := make([]any, 0, len(items))
				for _, item := range items {
					enum = append(enum, typedExample(strings.TrimSpace(item), schema["type"]))
				}
				schema["enum"] = enum
			case "minimum":
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					schema["minimum"] = v
				}
			case "maximum":
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					schema["maximum"] = v
				}
			case "minLength":
				if v, err := strconv.Atoi(val); err == nil {
					schema["minLength"] = v
				}
			case "maxLength":
				if v, err := strconv.Atoi(val); err == nil {
					schema["maxLength"] = v
				}
			case "pattern":
				schema["pattern"] = val
			}
		}
	}
}

func typedExample(raw string, typ any) any {
	kind, _ := typ.(string)
	switch kind {
	case "integer":
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return v
		}
	case "number":
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
	case "boolean":
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
	case "array", "object":
		var out any
		if err := json.Unmarshal([]byte(raw), &out); err == nil {
			return out
		}
	}
	return raw
}
