package utils

import (
	"log/slog"
	"strconv"
)

func CopySlice[T any](s []T) []T {
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func CopyMap[K comparable, V any](s map[K]V) map[K]V {
	out := make(map[K]V)
	for k, v := range s {
		out[k] = v
	}
	return out
}

func Ptr[T any](a T) *T {
	return &a
}

func Interface(data any) interface{} {
	return data
}

func UnwrapOrF(f *float64, or float64) float64 {
	if f == nil {
		return or
	}
	return *f
}

func Float(s string) float64 {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		slog.Error(
			"failed to parse as float64",
			slog.String("error", err.Error()),
			slog.String("str", s),
		)
	}
	return n
}
