package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestPactDamageStatMapStillEmitsTheNestedMod refuses if the producer this consumer was written for ever leaves the generated table again.
func TestPactDamageStatMapStillEmitsTheNestedMod(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		skill, stat, name string
		flags             mod.MFlag
		kw                mod.KeywordFlag
	}{
		{"PactOfBeidat", "pact_skill_damage_+%_final_with_hits_and_ailments_to_grant", "BeidatPactDamage",
			mod.MFlagSpell, mod.KeywordFlagHit | mod.KeywordFlagAilment},
		{"PactOfGhorr", "pact_skill_grant_damage_over_time_+%_final_to_spells", "GhorrPactDamage",
			mod.MFlagDot, 0},
		{"PactOfKtash", "pact_skill_grant_damage_+%_final_to_exerted_skills", "KtashPactDamage", 0, 0},
		{"PactOfLycia", "pact_skill_grant_damage_+%_final_to_exerted_skills", "LyciaPactDamage", 0, 0},
	} {
		t.Run(tc.skill, func(t *testing.T) {
			stat := tc.stat
			entry, ok := raw.SkillStatMaps[tc.skill][stat]
			if !ok {
				t.Fatalf("REFUSING: %s has no %q entry at all", tc.skill, stat)
			}
			if len(entry.Mods) != 1 {
				t.Fatalf("%s %q: %d mods, want 1 (an EMPTY entry is the omission this test "+
					"exists to catch -- it shadows the global map, so nothing else fails)",
					tc.skill, stat, len(entry.Mods))
			}
			m := entry.Mods[0]
			if m.Name() != tc.name || m.Type() != mod.TypeList {
				t.Fatalf("%s: got %s/%s, want %s/LIST", tc.skill, m.Name(), m.Type(), tc.name)
			}
			payload, ok := m.Value().List().(mod.PactDamage)
			if !ok {
				t.Fatalf("%s: payload is %T, want mod.PactDamage", tc.skill, m.Value().List())
			}
			if payload.Mod == nil {
				t.Fatalf("%s: mod.PactDamage carries no nested mod", tc.skill)
			}
			if payload.Mod.Name() != "Damage" || payload.Mod.Type() != mod.TypeMore {
				t.Fatalf("%s: nested mod is %s/%s, want Damage/MORE",
					tc.skill, payload.Mod.Name(), payload.Mod.Type())
			}
			if payload.Mod.Flags() != tc.flags || payload.Mod.KeywordFlags() != tc.kw {
				t.Fatalf("%s: nested mod flags=%#x kw=%#x, want flags=%#x kw=%#x",
					tc.skill, payload.Mod.Flags(), payload.Mod.KeywordFlags(), tc.flags, tc.kw)
			}
		})
	}
}
