package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestBloodMagicFamilyEmitsPinNames: The Blood Magic family had FIVE parser rows emitting names the pin does not have.
func TestBloodMagicFamilyEmitsPinNames(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		line      string
		name      string
		modFlags  mod.MFlag
		keyword   mod.KeywordFlag
		skillType string
	}{
		{"Attacks Cost Life instead of Mana", "CostLifeInsteadOfMana", mod.MFlagAttack, 0, ""},
		{"Skills which throw Traps Cost Life instead of Mana", "CostLifeInsteadOfMana", 0, 0, string(data.SkillTypeTrapped)},
		{"Curse Skills cost Life instead of Mana", "CostLifeInsteadOfMana", 0, mod.KeywordFlagCurse, ""},
		{"Your Skills that throw Mines reserve Life instead of Mana", "BloodMagicReserved", 0, mod.KeywordFlagMine, ""},
		{"Curse Aura Skills reserve Life instead of Mana", "BloodMagicReserved", 0, mod.KeywordFlagCurse, string(data.SkillTypeAura)},
	} {
		t.Run(tc.line, func(t *testing.T) {
			t.Parallel()

			e := ParseMod(tc.line, false)
			if e == nil || len(e.ModList) != 1 {
				t.Fatalf("ParseMod(%q) produced %v, want exactly one mod", tc.line, e)
			}
			if e.Extra != "" {
				t.Errorf("leftover Extra %q — the line would be discarded entirely", e.Extra)
			}
			m := e.ModList[0]
			if m.Name() != tc.name {
				t.Fatalf("mod name = %q, want %q", m.Name(), tc.name)
			}
			if m.Flags() != tc.modFlags {
				t.Errorf("ModFlags = %v, want %v", m.Flags(), tc.modFlags)
			}
			if m.KeywordFlags() != tc.keyword {
				t.Errorf("KeywordFlags = %v, want %v", m.KeywordFlags(), tc.keyword)
			}

			var gotSkillType string
			for _, tag := range m.Tags() {
				if st, ok := tag.(*mod.SkillTypeTag); ok && len(st.SkillTypeList) == 1 {
					gotSkillType = st.SkillTypeList[0]
				}
			}
			if gotSkillType != tc.skillType {
				t.Errorf("SkillType tag = %q, want %q", gotSkillType, tc.skillType)
			}
		})
	}
}

// TestRemovesAllManaKeepsBothMods: The `removes all mana` line carries TWO mods and only the second was wrong.
func TestRemovesAllManaKeepsBothMods(t *testing.T) {
	t.Parallel()

	e := ParseMod("Removes all Mana. Spend Life instead of Mana for Skills", false)
	if e == nil || len(e.ModList) != 2 {
		t.Fatalf("ParseMod produced %v, want two mods", e)
	}
	if e.Extra != "" {
		t.Errorf("leftover Extra %q", e.Extra)
	}
	names := []string{e.ModList[0].Name(), e.ModList[1].Name()}
	if names[0] != "Mana" || names[1] != "CostLifeInsteadOfMana" {
		t.Errorf("names = %v, want [Mana CostLifeInsteadOfMana] (ModParser.lua:2264)", names)
	}
}
