package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSupremeEgoLinesParseDespiteStaleModCache: Supreme Ego (`Tree:49639`) carries three lines, and PoB's DUMP shows only the third.
func TestSupremeEgoLinesParseDespiteStaleModCache(t *testing.T) {
	t.Run("aura effect per mana reserved", func(t *testing.T) {
		entry := ParseMod("Aura Skills have 1% more Aura Effect per 2% of maximum Mana they Reserve", false)
		if len(entry.ModList) != 1 {
			t.Fatalf("got %d mods, want 1", len(entry.ModList))
		}
		m := entry.ModList[0]
		if m.Name() != "AuraEffect" || m.Type() != mod.TypeMore {
			t.Fatalf("got %s %s, want AuraEffect MORE", m.Name(), m.Type())
		}

		var sawSkillType, sawPerStat bool
		for _, tag := range m.Tags() {
			switch tt := tag.(type) {
			case *mod.SkillTypeTag:
				_ = tt
				sawSkillType = true
			case *mod.PerStatTag:
				sawPerStat = true
				if tt.Divide != 2 {
					t.Errorf("PerStat Divide = %v, want 2", tt.Divide)
				}
				if len(tt.StatList) != 1 || tt.StatList[0] != "ManaReservedPercent" {
					t.Errorf("PerStat StatList = %v, want [ManaReservedPercent]", tt.StatList)
				}
			}
		}
		if !sawSkillType {
			t.Error("missing the SkillType(Aura) tag PoB's live parser emits")
		}
		if !sawPerStat {
			t.Error("missing the PerStat(ManaReservedPercent, div 2) tag — without it the mod is unconditional and wrong")
		}
	})

	t.Run("self auras only affect you", func(t *testing.T) {
		entry := ParseMod("Auras from your Skills can only affect you", false)
		if len(entry.ModList) != 1 {
			t.Fatalf("got %d mods, want 1", len(entry.ModList))
		}
		if got := entry.ModList[0].Name(); got != "SelfAurasOnlyAffectYou" {
			t.Fatalf("got %s, want SelfAurasOnlyAffectYou", got)
		}
	})
}
