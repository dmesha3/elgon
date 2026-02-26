package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// LoadEnv populates cfg from environment variables using struct tags.
// Tags:
// - env:"NAME" (required)
// - default:"value" (optional fallback)
// - required:"true" (optional strict requirement)
func LoadEnv[T any]() (T, error) {
	var cfg T
	err := loadFromEnv(reflect.ValueOf(&cfg).Elem())
	return cfg, err
}

// LoadJSONFile decodes JSON config into cfg with strict unknown-field checks.
func LoadJSONFile[T any](path string) (T, error) {
	var cfg T
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func loadFromEnv(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		ft := t.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Struct {
			if err := loadFromEnv(field); err != nil {
				return err
			}
			continue
		}

		envKey := strings.TrimSpace(ft.Tag.Get("env"))
		if envKey == "" {
			continue
		}
		raw := strings.TrimSpace(os.Getenv(envKey))
		if raw == "" {
			raw = strings.TrimSpace(ft.Tag.Get("default"))
		}
		req := ft.Tag.Get("required") == "true"
		if raw == "" && req {
			return fmt.Errorf("config: missing required env %s", envKey)
		}
		if raw == "" {
			continue
		}
		if err := setValue(field, raw); err != nil {
			return fmt.Errorf("config: env %s: %w", envKey, err)
		}
	}
	return nil
}

func setValue(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
		return nil
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(v)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type().PkgPath() == "time" && field.Type().Name() == "Duration" {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
			return nil
		}
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(v)
		return nil
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		field.SetFloat(v)
		return nil
	default:
		return fmt.Errorf("unsupported field kind %s", field.Kind())
	}
}
