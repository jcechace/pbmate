package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SetByPath sets a field on target (must be a pointer to struct) identified by
// a dot-separated path of yaml tag names. The rawValue string is coerced to
// the field's type. Supported leaf types: string, bool, *bool, int, int64.
// Map, slice, and struct leaves return an error.
func SetByPath(target any, dotPath string, rawValue string) error {
	field, err := resolveField(target, dotPath)
	if err != nil {
		return err
	}
	return setFieldValue(field, rawValue)
}

// GetByPath reads a field from target (must be a pointer to struct) identified
// by a dot-separated path of yaml tag names and returns its string
// representation.
func GetByPath(target any, dotPath string) (string, error) {
	field, err := resolveField(target, dotPath)
	if err != nil {
		return "", err
	}
	return formatFieldValue(field), nil
}

// UnsetByPath resets a field on target (must be a pointer to struct) to its
// zero value. For pointer types (e.g. *bool) this means nil; for value types
// it means the Go zero value ("", false, 0). Map, slice, and struct leaves
// return an error.
func UnsetByPath(target any, dotPath string) error {
	field, err := resolveField(target, dotPath)
	if err != nil {
		return err
	}

	switch field.Kind() {
	case reflect.Map, reflect.Slice, reflect.Struct:
		return fmt.Errorf("cannot unset composite key %q directly", dotPath)
	default:
		field.Set(reflect.Zero(field.Type()))
		return nil
	}
}

// resolveField walks the struct hierarchy following dot-separated yaml tag
// names and returns the final reflect.Value.
func resolveField(target any, dotPath string) (reflect.Value, error) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("target must be a pointer to struct")
	}
	v = v.Elem()

	segments := strings.Split(dotPath, ".")
	for i, seg := range segments {
		field, ok := findFieldByYAMLTag(v, seg)
		if !ok {
			return reflect.Value{}, fmt.Errorf("unknown key: %s", seg)
		}
		v = field
		// Dereference pointers only when there are more segments to walk
		// (nested structs). At the leaf, preserve the pointer so callers
		// can distinguish nil from zero (e.g. *bool nil vs false).
		if v.Kind() == reflect.Pointer && i < len(segments)-1 {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
	}
	return v, nil
}

// findFieldByYAMLTag finds a struct field whose yaml tag key matches name.
func findFieldByYAMLTag(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		if yamlTagName(t.Field(i)) == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// yamlTagName extracts the key name from a struct field's yaml tag,
// stripping modifiers like ",omitempty".
func yamlTagName(f reflect.StructField) string {
	tag := f.Tag.Get("yaml")
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// setFieldValue coerces rawValue to the field's type and sets it.
func setFieldValue(field reflect.Value, rawValue string) error {
	// Handle *bool specially — it's the only pointer type we support.
	if field.Kind() == reflect.Pointer && field.Type().Elem().Kind() == reflect.Bool {
		b, err := strconv.ParseBool(rawValue)
		if err != nil {
			return fmt.Errorf("invalid bool value: %q", rawValue)
		}
		field.Set(reflect.ValueOf(&b))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(rawValue)
	case reflect.Bool:
		b, err := strconv.ParseBool(rawValue)
		if err != nil {
			return fmt.Errorf("invalid bool value: %q", rawValue)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int value: %q", rawValue)
		}
		field.SetInt(n)
	case reflect.Map, reflect.Slice, reflect.Struct:
		return fmt.Errorf("cannot set composite key %q directly", rawValue)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Type())
	}
	return nil
}

// formatFieldValue returns a string representation of a field value.
func formatFieldValue(field reflect.Value) string {
	// Handle *bool.
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}
		return formatFieldValue(field.Elem())
	}

	switch field.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	case reflect.Int, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	default:
		return field.String()
	}
}
