package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestMultistrikeAppliesToMeleeNotRanged pins the two Multistrike rows that PoB 3.20's KeywordFlag/ModFlag renumbering silently corrupted.
func TestMultistrikeAppliesToMeleeNotRanged(t *testing.T) {
	const stat = "support_multiple_attacks_melee_attack_speed_+%_final"
	want := mod.MFlagWeaponMelee | mod.MFlagUnarmed

	for _, id := range []string{"SupportMultistrike", "SupportAwakenedMultistrike", "SupportGreaterMultistrike"} {
		entry := raw.SkillStatMaps[id][stat]
		if entry == nil {
			t.Fatalf("REFUSING: %s / %s is absent from SkillStatMaps — the row this test "+
				"exists for is gone, so a pass would mean nothing", id, stat)
		}

		found := 0
		for _, m := range entry.Mods {
			for _, tag := range m.Tags() {
				fo, ok := tag.(*mod.ModFlagOrTag)
				if !ok {
					continue
				}
				found++
				if fo.Flag != want {
					t.Errorf("%s / %s: ModFlagOr = %#x, PoB (sup_str.lua:3221) says %#x "+
						"(WeaponMelee|Unarmed)", id, stat, fo.Flag, want)
				}
			}
		}
		if found == 0 {
			t.Errorf("REFUSING: %s / %s carries no ModFlagOr tag at all", id, stat)
		}
	}
}
