package utils

import (
	"fmt"
	"log/slog"
	"reflect"
)

func Has[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

func HasTrue[K comparable, V any](m map[K]V, key K) bool {
	val, ok := m[key]
	if !ok {
		return false
	}

	if boolVal, ok := getBool(val); ok {
		return boolVal
	}

	slog.Warn(
		"failed to cast to bool",
		slog.Any("key", key),
		slog.Any("value", val),
	)

	return false
}

func getBool(value interface{}) (bool, bool) {
	val, ok := value.(bool)
	return val, ok
}

func GetOr[V any](m any, key string, or V) V {
	r := reflect.ValueOf(m)

	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}

	if r.IsZero() {
		return or
	}

	f := r.FieldByName(key)
	if f.Kind() == reflect.Invalid {
		slog.Error("invalid key", slog.String("key", key), slog.String("obj", fmt.Sprintf("%#v", m)))
		return or
	}

	if f.IsZero() {
		return or
	}

	return f.Interface().(V)
}

func Set(m any, key string, value any) {
	r := reflect.ValueOf(m)

	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}

	f := r.FieldByName(key)
	if f.Kind() == reflect.Invalid {
		slog.Error("invalid key", slog.String("key", key), slog.String("obj", fmt.Sprintf("%#v", m)))
		return
	}

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Float64 && f.Kind() == reflect.Bool {
		f.Set(reflect.ValueOf(value.(float64) == 1))
	} else if v.Kind() == reflect.Float64 && f.Kind() == reflect.Int {
		f.Set(reflect.ValueOf(int(value.(float64))))
	} else {
		f.Set(v)
	}
}

func MissingOrFalse[K comparable, V any](m map[K]V, key K) bool {
	val, ok := m[key]
	if !ok {
		return true
	}

	if boolVal, ok := getBool(val); ok {
		return !boolVal
	}

	slog.Warn(
		"failed to cast to bool",
		slog.Any("key", key),
		slog.Any("value", val),
	)

	return false
}

func MapConcat[T comparable, M any](maps ...map[T]M) map[T]M {
	out := make(map[T]M)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
