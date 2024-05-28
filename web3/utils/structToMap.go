package utils

import (
	"reflect"
	"strings"
)

// StructToMap converts a struct to a map[string]interface{}
func StructToMap(obj any) map[string]any {
	result := make(map[string]any)
	val := reflect.ValueOf(obj)
	typ := val.Type()
	if val.Kind() == reflect.Map {
		for _, key := range val.MapKeys() {
			if key.Kind() == reflect.String {
				result[key.Interface().(string)] = val.MapIndex(key).Interface()
			}
		}
		return result
	}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := typ.Field(i).Tag.Get("json")

		// Skip unexported fields or fields without a json tag
		if field.CanInterface() && tag != "" {
			// Check if the field is a nil pointer
			if field.Kind() == reflect.Ptr && field.IsNil() {
				continue
			}
			parts := strings.Split(tag, ",")
			if containsOmitEmpty(parts) {
				if field.Kind() == reflect.Slice && field.Len() == 0 {
					continue
				} else if isIntKind(field.Kind()) && field.Uint() == 0 {
					continue
				} else if field.Kind() == reflect.Bool && field.Bool() == false {
					continue
				} else if field.Kind() == reflect.String && field.Len() == 0 {
					continue
				}
			}
			if len(parts) > 0 {
				result[parts[0]] = field.Interface()
			}
		}
	}
	return result
}

func isIntKind(kind reflect.Kind) bool {
	if kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 {
		return true
	}
	return false
}

func containsOmitEmpty(input []string) bool {
	for _, item := range input {
		if item == "omitempty" {
			return true
		}
	}
	return false
}
