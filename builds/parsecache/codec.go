package parsecache

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"reflect"
	"sync"
)

const (
	magic         = "GPBC"
	formatVersion = 1
)

type bitset struct {
	b []byte
	n int
}

func (s *bitset) push(v bool) {
	if s.n%8 == 0 {
		s.b = append(s.b, 0)
	}
	if v {
		s.b[s.n/8] |= 1 << (s.n % 8)
	}
	s.n++
}

func (s *bitset) next() bool {
	i := s.n
	s.n++
	if i/8 >= len(s.b) {
		return false
	}
	return s.b[i/8]&(1<<(i%8)) != 0
}

func walk(v reflect.Value, out *bitset, rec *bitset) {
	switch v.Kind() {
	case reflect.Pointer:
		if rec == nil {
			out.push(!v.IsNil())
		} else if rec.next() && v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if !v.IsNil() {
			walk(v.Elem(), out, rec)
		}
	case reflect.Slice:
		if rec == nil {
			out.push(!v.IsNil())
		} else if rec.next() && v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		}
		for i := range v.Len() {
			walk(v.Index(i), out, rec)
		}
	case reflect.Struct:
		for _, i := range structPlan(v.Type()) {
			walk(v.Field(i), out, rec)
		}
	default:
	}
}

var structPlans sync.Map

func structPlan(t reflect.Type) []int {
	if p, ok := structPlans.Load(t); ok {
		return p.([]int)
	}
	var plan []int
	for i := range t.NumField() {
		f := t.Field(i)
		if f.IsExported() && mayHoldPresence(f.Type) {
			plan = append(plan, i)
		}
	}
	structPlans.Store(t, plan)
	return plan
}

func mayHoldPresence(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice:
		return true
	case reflect.Array:
		return mayHoldPresence(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if t.Field(i).IsExported() && mayHoldPresence(t.Field(i).Type) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func Encode(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil, fmt.Errorf("parsecache: Encode wants a non-nil pointer, got %T", v)
	}

	var presence bitset
	walk(rv.Elem(), &presence, nil)

	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(v); err != nil {
		return nil, fmt.Errorf("parsecache: gob encode: %w", err)
	}

	out := make([]byte, 0, len(magic)+1+binary.MaxVarintLen64*2+len(presence.b)+payload.Len())
	out = append(out, magic...)
	out = append(out, formatVersion)
	//nolint:gosec // G115: both are lengths of slices this function just built, so neither can be
	out = binary.AppendUvarint(out, uint64(presence.n))
	out = binary.AppendUvarint(out, uint64(len(presence.b))) //nolint:gosec // G115: a slice length.
	out = append(out, presence.b...)
	out = append(out, payload.Bytes()...)
	return out, nil
}

func Decode(blob []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("parsecache: Decode wants a non-nil pointer, got %T", v)
	}
	if len(blob) < len(magic)+1 || string(blob[:len(magic)]) != magic {
		return fmt.Errorf("parsecache: not a %s blob", magic)
	}
	if blob[len(magic)] != formatVersion {
		return fmt.Errorf("parsecache: format version %d, want %d", blob[len(magic)], formatVersion)
	}
	rest := blob[len(magic)+1:]

	bits, n := binary.Uvarint(rest)
	if n <= 0 {
		return fmt.Errorf("parsecache: truncated presence count")
	}
	rest = rest[n:]
	if bits > math.MaxInt {
		return fmt.Errorf("parsecache: presence count %d is not representable", bits)
	}
	size, n := binary.Uvarint(rest)
	if n <= 0 || uint64(len(rest[n:])) < size { //nolint:gosec // G115: a slice length.
		return fmt.Errorf("parsecache: truncated presence bitmap")
	}
	rest = rest[n:]
	rec := bitset{b: rest[:size]}
	rest = rest[size:]

	if err := gob.NewDecoder(bytes.NewReader(rest)).Decode(v); err != nil {
		return fmt.Errorf("parsecache: gob decode: %w", err)
	}

	walk(rv.Elem(), nil, &rec)
	if rec.n != int(bits) {
		return fmt.Errorf("parsecache: presence walk visited %d nodes, blob records %d", rec.n, bits)
	}
	return nil
}
