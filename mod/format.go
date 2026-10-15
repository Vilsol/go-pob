package mod

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

//pob:src/Modules/ModTools.lua:196-225
func FormatValue(v any) string {
	return formatValue(reflect.ValueOf(v), 0)
}

//pob:src/Modules/ModTools.lua:151-186, src/Modules/ModTools.lua:196-225
func formatValue(v reflect.Value, depth int) string {
	if !v.IsValid() {
		return "nil"
	}

	if depth > 8 {
		return "..."
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return formatValue(v.Elem(), depth+1)
	case reflect.Slice, reflect.Array:
		parts := make([]string, v.Len())
		for i := range parts {
			parts[i] = formatValue(v.Index(i), depth+1)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case reflect.Struct:
		t := v.Type()
		parts := make([]string, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			parts = append(parts, t.Field(i).Name+"="+formatValue(v.Field(i), depth+1))
		}
		sort.Strings(parts)
		return "{" + strings.Join(parts, "/") + "}"
	case reflect.Float64, reflect.Float32:
		return fmt.Sprintf("%.12g", v.Float())
	default:
		return fmt.Sprintf("%v", v)
	}
}
