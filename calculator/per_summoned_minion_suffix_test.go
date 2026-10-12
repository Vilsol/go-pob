package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPerSummonedMinionSuffixesCarryPerStat pins three modifier-tag rows the pin has and this tree did not.
func TestPerSummonedMinionSuffixesCarryPerStat(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
		typ  mod.Type
		val  float64
		stat string
	}{
		{"2% increased Attack and Cast Speed per Summoned Raging Spirit", "Speed", mod.TypeIncrease, 2, "ActiveRagingSpiritLimit"},
		{"15% increased Attack Damage per Raised Zombie", "Damage", mod.TypeIncrease, 15, "ActiveZombieLimit"},
		{"10% increased Defences per Raised Spectre", "Defences", mod.TypeIncrease, 10, "ActiveSpectreLimit"},
		{"2% increased Movement Speed per Summoned Phantasm", "MovementSpeed", mod.TypeIncrease, 2, "ActivePhantasmLimit"},
		{"6% increased Spell Damage per Summoned Skeleton", "Damage", mod.TypeIncrease, 6, "ActiveSkeletonLimit"},

		{"1% increased Damage per Summoned Void Spawn", "Damage", mod.TypeIncrease, 1, "ActiveVoidSpawnLimit"},
	} {
		t.Run(tc.stat, func(t *testing.T) {
			entry := ParseMod(strings.ToLower(tc.line), false)
			if entry == nil || len(entry.ModList) != 1 {
				got := 0
				if entry != nil {
					got = len(entry.ModList)
				}
				t.Fatalf("%q: got %d mods, want 1", tc.line, got)
			}
			m := entry.ModList[0]
			if m.Name() != tc.name || m.Type() != tc.typ || m.Value().Float() != tc.val {
				t.Fatalf("%q: got %s/%s/%v, want %s/%s/%v",
					tc.line, m.Name(), m.Type(), m.Value(), tc.name, tc.typ, tc.val)
			}
			found := false
			for _, tag := range m.Tags() {
				ps, ok := tag.(*mod.PerStatTag)
				if ok && slices.Contains(ps.StatList, tc.stat) {
					found = true
				}
			}
			if !found {
				t.Errorf("%q: no PerStat tag on %s (want stat %s); a per-line with no PerStat tag "+
					"applies its base value once instead of once per minion", tc.line, tc.name, tc.stat)
			}
		})
	}
}
