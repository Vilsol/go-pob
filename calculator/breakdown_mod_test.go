package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestBreakdownModEmitsOnEitherTerm pins CalcBreakdown.lua:72, `if inc ~= 0 or more ~= 1`.
func TestBreakdownModEmitsOnEitherTerm(t *testing.T) {
	cases := []struct {
		name string
		mods []mod.Mod
		want bool
	}{
		{name: "neither term", mods: nil, want: false},
		{name: "inc only", mods: []mod.Mod{MOD("Test", "INC", 20)}, want: true},
		{name: "more only", mods: []mod.Mod{MOD("Test", "MORE", 30)}, want: true},
		{name: "both", mods: []mod.Mod{MOD("Test", "INC", 20), MOD("Test", "MORE", 30)}, want: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := moddb.NewModDB()
			for _, m := range c.mods {
				db.AddMod(m)
			}
			b := NewBreakdown(db, map[string]float64{}, nil)
			b.Mod("TestMod", db, nil, "Test")
			if got := b.Has("TestMod"); got != c.want {
				t.Fatalf("breakdown emitted %v, want %v", got, c.want)
			}
		})
	}
}
