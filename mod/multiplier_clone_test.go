package mod

import (
	"reflect"
	"testing"
)

// TestMultiplierTagCloneCopiesEveryField is a REFLECTIVE guard.
func TestMultiplierTagCloneCopiesEveryField(t *testing.T) {
	src := Multiplier("A", "B")

	v := reflect.ValueOf(src).Elem()
	typ := v.Type()
	filled := 0
	for i := 0; i < typ.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		fillDistinguishable(t, typ.Field(i).Name, f)
		filled++
	}
	if filled != typ.NumField() {
		t.Fatalf("REFUSING: filled %d of %d fields; an unfilled field cannot be graded",
			filled, typ.NumField())
	}

	clone, ok := src.Clone().(*MultiplierTag)
	if !ok {
		t.Fatal("REFUSING: Clone did not return a *MultiplierTag")
	}

	cv := reflect.ValueOf(clone).Elem()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		want, got := v.Field(i).Interface(), cv.Field(i).Interface()
		if !reflect.DeepEqual(want, got) {
			t.Errorf("Clone dropped field %s: want %v, got %v (deref: %s)",
				name, want, got, derefString(cv.Field(i)))
		}
	}
}

func fillDistinguishable(t *testing.T, name string, f reflect.Value) {
	t.Helper()
	switch f.Kind() {
	case reflect.String:
		f.SetString("clone-probe-" + name)
	case reflect.Bool:
		f.SetBool(true)
	case reflect.Float64:
		f.SetFloat(7.5)
	case reflect.Slice:
		f.Set(reflect.MakeSlice(f.Type(), 1, 1))
		fillDistinguishable(t, name, f.Index(0))
	case reflect.Pointer:
		f.Set(reflect.New(f.Type().Elem()))
		fillDistinguishable(t, name, f.Elem())
	default:
		t.Fatalf("REFUSING: no distinguishable value for field %s of kind %s -- extend this "+
			"helper rather than letting the field go ungraded", name, f.Kind())
	}
}

func derefString(f reflect.Value) string {
	if f.Kind() == reflect.Pointer && !f.IsNil() {
		return "non-nil"
	}
	if f.Kind() == reflect.Pointer {
		return "nil"
	}
	return "n/a"
}
