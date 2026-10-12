package raw

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillTypeListRowsAreNotEmpty pins the two SkillStatMap entries that the global emitter used to drop.
func TestSkillTypeListRowsAreNotEmpty(t *testing.T) {
	for _, tc := range []struct {
		key     string
		dataKey string
		types   []string
	}{
		{"summon_sacred_wisps_on_hit", "triggeredBySacredWisps", []string{"Spell", "RangedAttack"}},
		{"support_autocast_instant_spells", "triggeredByAutomation", []string{"Triggerable", "Spell", "Instant"}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			entry, ok := SkillStatMap[tc.key]
			if !ok {
				t.Fatalf("REFUSING: %q is not in SkillStatMap at all — an absent key and an "+
					"empty one are different defects and this must not report them the same way",
					tc.key)
			}
			testza.AssertEqual(t, 1, len(entry.Mods),
				"%s: the pin maps this stat to exactly one mod", tc.key)
			if len(entry.Mods) != 1 {
				return
			}

			var got *mod.SkillTypeTag
			for _, tag := range entry.Mods[0].Tags() {
				if st, ok := tag.(*mod.SkillTypeTag); ok {
					got = st
				}
			}
			if got == nil {
				t.Fatalf("%s: the mod carries no SkillType tag, so it would apply to EVERY skill "+
					"— over-application replacing under-application, which is worse than the "+
					"empty list this replaced", tc.key)
			}
			testza.AssertEqual(t, tc.types, got.SkillTypeList,
				"%s: skillTypeList differs from the pin", tc.key)
			testza.AssertFalse(t, got.Negative, "%s: the pin's tag is not negated", tc.key)
		})
	}
}
