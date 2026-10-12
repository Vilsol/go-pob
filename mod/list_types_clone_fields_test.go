package mod

import (
	"reflect"
	"testing"
)

// TestCloneCarriesEveryField asserts that every payload type's Clone preserves every exported scalar field it declares.
func TestCloneCarriesEveryField(t *testing.T) {
	checked := 0
	for name, inst := range payloadInstances(t) {
		typ := reflect.TypeOf(inst)
		if typ.Kind() != reflect.Struct {
			continue
		}

		filled := reflect.New(typ).Elem()
		var scalars []int
		for i := 0; i < typ.NumField(); i++ {
			f := filled.Field(i)
			if !f.CanSet() {
				continue
			}
			switch f.Kind() {
			case reflect.Bool:
				f.SetBool(true)
			case reflect.String:
				f.SetString("sentinel")
			case reflect.Float32, reflect.Float64:
				f.SetFloat(7)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				f.SetInt(7)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				f.SetUint(7)
			case reflect.Interface:
				if reflect.TypeOf(NewFloat("Scaffold", TypeBase, 1)).Implements(f.Type()) {
					f.Set(reflect.ValueOf(NewFloat("Scaffold", TypeBase, 1)))
				}
				continue
			default:
				continue
			}
			scalars = append(scalars, i)
		}
		if len(scalars) == 0 {
			continue
		}

		var cloned Cloneable
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s.Clone panicked on a fully-populated value (%v). The scaffolding "+
						"above does not build one of its field shapes, so this type is NOT "+
						"checked -- extend the switch rather than skipping it.", name, r)
				}
			}()
			cloned = filled.Interface().(Cloneable).Clone(false)
		}()
		if cloned == nil {
			continue
		}
		got := reflect.ValueOf(cloned)
		if got.Kind() == reflect.Pointer {
			got = got.Elem()
		}
		if got.Type() != typ {
			t.Errorf("%s.Clone(false) returned %s, want %s", name, got.Type(), typ)
			continue
		}
		for _, i := range scalars {
			want, have := filled.Field(i), got.Field(i)
			if !reflect.DeepEqual(want.Interface(), have.Interface()) {
				t.Errorf("%s.Clone DROPPED field %s: set %v, got back %v.\n"+
					"Clone is a hand-written struct literal -- add the field to it. A dropped "+
					"scalar reads as 'the producer never set it', not as a bug.",
					name, typ.Field(i).Name, want.Interface(), have.Interface())
			}
			checked++
		}
	}

	if checked == 0 {
		t.Fatal("checked 0 fields -- the guard is vacuous, not clean")
	}
	t.Logf("checked %d scalar fields across the payload registry", checked)
}
