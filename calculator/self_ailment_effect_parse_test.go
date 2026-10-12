package calculator

import (
	"slices"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSelfNonDamagingAilmentEffectLinesParse pins ModParser.lua:2888 and :787, the two rows that reduce the effect of non-damaging ailments ON YOU.
func TestSelfNonDamagingAilmentEffectLinesParse(t *testing.T) {
	wantSix := []string{
		"SelfBrittleEffect", "SelfChillEffect", "SelfFreezeEffect",
		"SelfSapEffect", "SelfScorchEffect", "SelfShockEffect",
	}

	for _, tc := range []struct {
		line   string
		names  []string
		value  float64
		tagged bool
	}{
		{"Non-Damaging Ailments have 50% reduced Effect on you while you have Arcane Surge", wantSix, -50, true},
		{"25% reduced Effect of Non-Damaging Ailments on you", wantSix, -25, false},
		{"20% reduced Effect of Chill on you", []string{"SelfChillEffect"}, -20, false},
		{"20% reduced Effect of Non-Damaging Ailments", []string{
			"EnemyBrittleEffect", "EnemyChillEffect", "EnemyFreezeEffect",
			"EnemySapEffect", "EnemyScorchEffect", "EnemyShockEffect",
		}, -20, false},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("ParseMod(%q) returned nil", tc.line)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — processModLine discards the whole line", e.Extra)
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
				if m.Type() != mod.TypeIncrease {
					t.Errorf("%s type = %v, want INC", m.Name(), m.Type())
				}
				if v := m.Value().Float(); v != tc.value {
					t.Errorf("%s value = %v, want %v", m.Name(), v, tc.value)
				}
				hasCond := false
				for _, tg := range m.Tags() {
					if c, ok := tg.(*mod.ConditionTag); ok && slices.Contains(c.VarList, "AffectedByArcaneSurge") {
						hasCond = true
					}
				}
				if hasCond != tc.tagged {
					t.Errorf("%s AffectedByArcaneSurge tag = %v, want %v", m.Name(), hasCond, tc.tagged)
				}
			}
			sort.Strings(got)
			if len(got) != len(tc.names) {
				t.Fatalf("names = %v, want %v", got, tc.names)
			}
			for i := range got {
				if got[i] != tc.names[i] {
					t.Fatalf("names = %v, want %v", got, tc.names)
				}
			}
		})
	}
}
