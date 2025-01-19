package utils

import (
	"go/types"
	"log/slog"
	"strconv"

	"github.com/Vilsol/go-pob/mod"
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
		slog.Warn(
			"failed to parse as float64",
			slog.String("error", err.Error()),
			slog.String("str", s),
		)
	}
	return n
}

func Int(s string) int {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		slog.Warn(
			"failed to parse as float64",
			slog.String("error", err.Error()),
			slog.String("str", s),
		)
	}
	return int(n)
}

func Ternary[T any](cond bool, ifTrue T, ifFalse T) T {
	if cond {
		return ifTrue
	}

	return ifFalse
}

type numberLike interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64
}

func Number[T numberLike](val any) T {
	if val == nil {
		return T(0)
	}

	switch x := val.(type) {
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return T(f)
	case bool:
		if x {
			return 1
		}
		return 0
	case int:
		return T(x)
	case int8:
		return T(x)
	case int16:
		return T(x)
	case int32:
		return T(x)
	case int64:
		return T(x)
	case uint:
		return T(x)
	case uint8:
		return T(x)
	case uint16:
		return T(x)
	case uint32:
		return T(x)
	case uint64:
		return T(x)
	case uintptr:
		return T(x)
	case float32:
		return T(x)
	case float64:
		return T(x)
	case types.Nil:
		return T(0)
	}

	panic("unreachable")
}

func Or(val *mod.ModValueMulti, or float64) float64 {
	if val == nil {
		return or
	}

	if val.Type() != mod.ModValueMultiTypeFloat {
		return or
	}

	if val.Float() == 0 {
		return or
	}

	return val.Float()
}

func OrDefault(n float64, def float64) float64 {
	if n != 0 {
		return n
	}
	return def
}

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}
