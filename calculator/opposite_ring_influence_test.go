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
		{"70% increased Attack Damage if your opposite Ring is a Shaper Item", "shaper"},
		{"70% increased Attack Damage if your other Ring is a Shaper Item", "shaper"},
		{"80% increased Spell Damage if your opposite Ring is an Elder Item", "elder"},
		{"80% increased Spell Damage if your other Ring is an Elder Item", "elder"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want exactly 1 mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Fatalf("leftover Extra %q — the whole mod line is discarded", e.Extra)
			}

			var tag *mod.ItemConditionTag
			for _, tg := range e.ModList[0].Tags() {
				if c, ok := tg.(*mod.ItemConditionTag); ok {
					tag = c
				}
			}
			if tag == nil {
				t.Fatalf("no ItemCondition tag on %q", tc.line)
			}
			if tag.ItemSlot != "Ring {OtherSlotNum}" {
				t.Errorf("itemSlot = %q, want \"Ring {OtherSlotNum}\"; item setup resolves the placeholder to the OTHER ring, "+
					"so a literal slot would read the wrong one", tag.ItemSlot)
			}
			if got := map[string]bool{"shaper": tag.ShaperCond, "elder": tag.ElderCond}; !got[tc.want] || got["shaper"] == got["elder"] {
				t.Errorf("shaperCond=%v elderCond=%v, want only %sCond", tag.ShaperCond, tag.ElderCond, tc.want)
			}
		})
	}
}
