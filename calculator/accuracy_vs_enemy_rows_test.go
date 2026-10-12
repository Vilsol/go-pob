package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAccuracyVsEnemyRowsAreComplete pins ModParser.lua:2467-2469 as a FAMILY.
func TestAccuracyVsEnemyRowsAreComplete(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		line    string
		wantVal float64
		wantTyp mod.Type
		wantTag string
	}{
		{"40% more Accuracy Rating against Unique Enemies", 40, mod.TypeMore, "ActorCondition"},
		{"20% increased Accuracy Rating against Unique Enemies", 20, mod.TypeIncrease, "ActorCondition"},
		{"30% more Accuracy Rating at Close Range", 30, mod.TypeMore, "Condition"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e.Extra != "" {
				t.Fatalf("leftover %q -- an unparsed remainder DISCARDS THE WHOLE LINE", e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("got %d mods, want 1: %+v", len(e.ModList), e.ModList)
			}
			m := e.ModList[0]
			if m.Name() != "AccuracyVsEnemy" {
				t.Fatalf("name = %q, want AccuracyVsEnemy (a plain Accuracy mod applies to every enemy)", m.Name())
			}
			if m.Type() != tc.wantTyp || m.Value().Float() != tc.wantVal {
				t.Fatalf("type/value = %v/%v, want %v/%v", m.Type(), m.Value().Float(), tc.wantTyp, tc.wantVal)
			}
			if len(m.Tags()) != 1 {
				t.Fatalf("tags = %d, want exactly 1 (%s) -- an untagged mod applies UNCONDITIONALLY", len(m.Tags()), tc.wantTag)
			}
		})
	}
}
