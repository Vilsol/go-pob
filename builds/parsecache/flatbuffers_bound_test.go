package parsecache

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// BenchmarkStringMaterialisationFloor is the FlatBuffers evaluation, and it is a bound rather than an implementation on purpose.
func BenchmarkStringMaterialisationFloor(b *testing.B) {
	parsed := sampleParsed(b)
	type layout struct {
		buf   []byte
		spans [][2]int
	}
	layouts := make([]layout, len(parsed))
	for i, p := range parsed {
		var l layout
		forEachString(reflect.ValueOf(p).Elem(), func(s string) {
			l.spans = append(l.spans, [2]int{len(l.buf), len(l.buf) + len(s)})
			l.buf = append(l.buf, s...)
		})
		layouts[i] = l
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		l := layouts[i%len(layouts)]
		out := make([]string, len(l.spans))
		for j, sp := range l.spans {
			out[j] = string(l.buf[sp[0]:sp[1]])
		}
		if len(out) == 0 {
			b.Fatal("no strings")
		}
	}
}

func forEachString(v reflect.Value, fn func(string)) {
	switch v.Kind() {
	case reflect.String:
		fn(v.String())
	case reflect.Pointer:
		if !v.IsNil() {
			forEachString(v.Elem(), fn)
		}
	case reflect.Slice:
		for i := range v.Len() {
			forEachString(v.Index(i), fn)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				forEachString(v.Field(i), fn)
			}
		}
	default:
	}
}

// TestFlatbuffersPortSurface prices the port the bound above is compared against.
func TestFlatbuffersPortSurface(t *testing.T) {
	tables, fields, optionalScalars, stringFields, vectors := 0, 0, 0, 0, 0
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type)
	visit = func(ty reflect.Type) {
		if seen[ty] {
			return
		}
		seen[ty] = true
		switch ty.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visit(ty.Elem())
		case reflect.Struct:
			tables++
			for i := range ty.NumField() {
				f := ty.Field(i)
				if !f.IsExported() || f.Type.Kind() == reflect.Interface {
					continue
				}
				fields++
				switch f.Type.Kind() {
				case reflect.Pointer:
					optionalScalars++
				case reflect.Slice:
					vectors++
				case reflect.String:
					stringFields++
				default:
				}
				visit(f.Type)
			}
		default:
		}
	}
	visit(reflect.TypeOf(pob.PathOfBuilding{}))

	t.Logf("a flatbuffers port covers %d tables, %d fields — %d vectors, %d strings and "+
		"%d optional scalars needing explicit presence — plus a hand-written conversion layer in "+
		"both directions, because every consumer takes *pob.PathOfBuilding",
		tables, fields, vectors, stringFields, optionalScalars)
}
