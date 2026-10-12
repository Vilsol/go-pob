package utils

import (
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
)

var foldIndexCache sync.Map

const ambiguousField = -1

func foldIndex(t reflect.Type) map[string]int {
	if cached, ok := foldIndexCache.Load(t); ok {
		return cached.(map[string]int)
	}

	m := make(map[string]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		lower := strings.ToLower(f.Name)
		if _, dup := m[lower]; dup {
			m[lower] = ambiguousField
			continue
		}

		m[lower] = i
	}

	foldIndexCache.Store(t, m)

	return m
}

type fieldKey struct {
	t   reflect.Type
	key string
}

var fieldIndexCache sync.Map

func resolveField(r reflect.Value, key string) reflect.Value {
	k := fieldKey{r.Type(), key}
	cached, ok := fieldIndexCache.Load(k)
	if !ok {
		var index []int
		if sf, found := k.t.FieldByName(key); found {
			index = sf.Index
		}
		cached, _ = fieldIndexCache.LoadOrStore(k, index)
	}
	if index := cached.([]int); index != nil {
		if f := r.FieldByIndex(index); f.Kind() != reflect.Invalid {
			return f
		}
	}

	if r.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	idx, ok := foldIndex(r.Type())[strings.ToLower(key)]
	if !ok || idx == ambiguousField {
		return reflect.Value{}
	}

	return r.Field(idx)
}

type DroppedKey struct {
	Op   string
	Key  string
	Type string
}

var (
	droppedMu    sync.Mutex
	droppedCount = map[DroppedKey]int{}
)

func recordDropped(op string, key string, t reflect.Type) {
	name := "<nil>"
	if t != nil {
		name = t.String()
	}

	dk := DroppedKey{Op: op, Key: key, Type: name}

	droppedMu.Lock()
	first := droppedCount[dk] == 0
	droppedCount[dk]++
	droppedMu.Unlock()

	if first {
		slog.Debug("invalid key", slog.String("op", op), slog.String("key", key), slog.String("type", name))
	}
}

func DroppedKeys() (int, []struct {
	DroppedKey
	Count int
}) {
	droppedMu.Lock()
	defer droppedMu.Unlock()

	total := 0
	out := make([]struct {
		DroppedKey
		Count int
	}, 0, len(droppedCount))
	for k, c := range droppedCount {
		total += c
		out = append(out, struct {
			DroppedKey
			Count int
		}{k, c})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Op < out[j].Op
	})

	return total, out
}

func ResetDroppedKeys() {
	droppedMu.Lock()
	defer droppedMu.Unlock()
	droppedCount = map[DroppedKey]int{}
}

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

	f := resolveField(r, key)
	if f.Kind() == reflect.Invalid {
		recordDropped("get", key, r.Type())
		return or
	}

	if f.IsZero() {
		return or
	}

	if v, ok := f.Interface().(V); ok {
		return v
	}

	if f.Kind() == reflect.Pointer {
		if v, ok := f.Elem().Interface().(V); ok {
			return v
		}
	}

	recordDropped("get", key, r.Type())

	return or
}

func Set(m any, key string, value any) {
	r := reflect.ValueOf(m)

	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}

	f := resolveField(r, key)
	if f.Kind() == reflect.Invalid || !f.CanSet() {
		recordDropped("set", key, r.Type())
		return
	}

	v := reflect.ValueOf(value)
	switch {
	case v.Kind() == reflect.Float64 && f.Kind() == reflect.Bool:
		f.SetBool(value.(float64) == 1)
	case v.Kind() == reflect.Float64 && f.Kind() == reflect.Int:
		f.SetInt(int64(value.(float64)))
	case v.IsValid() && f.Kind() == reflect.Pointer && v.Type().AssignableTo(f.Type().Elem()):
		p := reflect.New(f.Type().Elem())
		p.Elem().Set(v)
		f.Set(p)
	case v.IsValid() && v.Type().AssignableTo(f.Type()):
		f.Set(v)
	default:
		recordDropped("set", key, r.Type())
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
