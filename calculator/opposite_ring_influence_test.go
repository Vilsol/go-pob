package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestOppositeRingInfluenceRowsAcceptBothSpellings pins ModParser.lua:1593-1594.
func TestOppositeRingInfluenceRowsAcceptBothSpellings(t *testing.T) {
	for _, tc := range []struct {
		line string
		want string
	}{
		{"70% increased Attack Damage if your opposite Ring is a Shaper Item", "ShaperItemInRing {OtherSlotNum}"},
		{"70% increased Attack Damage if your other Ring is a Shaper Item", "ShaperItemInRing {OtherSlotNum}"},
		{"80% increased Spell Damage if your opposite Ring is an Elder Item", "ElderItemInRing {OtherSlotNum}"},
		{"80% increased Spell Damage if your other Ring is an Elder Item", "ElderItemInRing {OtherSlotNum}"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want exactly 1 mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Fatalf("leftover Extra %q — the whole mod line is discarded", e.Extra)
			}

			var got []string
			for _, tg := range e.ModList[0].Tags() {
				if c, ok := tg.(*mod.ConditionTag); ok {
					got = append(got, c.VarList...)
				}
			}
			found := false
			for _, v := range got {
				if v == tc.want {
					found = true
				}
			}
			if !found {
				t.Errorf("condition vars = %v, want %q; the {OtherSlotNum} placeholder is what "+
					"item setup resolves to the OTHER ring, so a literal slot would read the wrong one", got, tc.want)
			}
		})
	}
}
