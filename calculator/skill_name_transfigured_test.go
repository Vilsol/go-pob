package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillNameLoopTagsIncludeTransfigured pins ModParser.lua:6046-6088: EVERY SkillName tag the per-gem loop builds carries includeTransfigured = true.
func TestSkillNameLoopTagsIncludeTransfigured(t *testing.T) {
	for _, line := range []string{
		"Lightning Tendrils deals 40% more Damage",
		"Barrage fires an additional Projectile",
		"Ethereal Knives Pierces an additional Target",
		"Barrage fires 2 additional Arrows",
		"Frost Blades has 10% increased Projectile Speed",
	} {
		e := ParseMod(line, false)
		if e == nil || e.Extra != "" || len(e.ModList) == 0 {
			t.Errorf("%q: ParseMod = %v, want a whole parse", line, e)
			continue
		}
		found := false
		for _, m := range e.ModList {
			for _, tag := range m.Tags() {
				if sn, ok := tag.(*mod.SkillNameTag); ok {
					found = true
					if !sn.IncludeTransfigured {
						t.Errorf("%q: SkillName tag %+v lacks includeTransfigured", line, *sn)
					}
				}
			}
		}
		if !found {
			t.Errorf("%q: no SkillName tag emitted", line)
		}
	}
}
