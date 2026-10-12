package utils

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOrKeepsZeroOverride pins Or against Lua's `or`.
func TestOrKeepsZeroOverride(t *testing.T) {
	zero := mod.NewModValueFloat(0)
	if got := Or(zero, 2907); got != 0 {
		t.Errorf("Or(override 0, fallback 2907) = %v, want 0 — Lua's `or` takes a 0 "+
			"override because 0 is truthy in Lua", got)
	}

	nonZero := mod.NewModValueFloat(42)
	if got := Or(nonZero, 2907); got != 42 {
		t.Errorf("Or(override 42, fallback 2907) = %v, want 42", got)
	}

	if got := Or(nil, 2907); got != 2907 {
		t.Errorf("Or(nil, 2907) = %v, want 2907 — an absent override is Lua nil, which "+
			"IS falsy", got)
	}
}
