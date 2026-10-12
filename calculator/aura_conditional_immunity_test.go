package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAuraConditionalSelfProtectionRowsParse pins ModParser.lua:4836-4838 and :4915-4921.
func TestAuraConditionalSelfProtectionRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line      string
		name      string
		skillName string
		condition string
	}{
		{"Unaffected by Vulnerability while affected by Determination", "CurseEffectOnSelf", "Vulnerability", "AffectedByDetermination"},
		{"Unaffected by Enfeeble while affected by Grace", "CurseEffectOnSelf", "Enfeeble", "AffectedByGrace"},
		{"Unaffected by Temporal Chains while affected by Haste", "CurseEffectOnSelf", "Temporal Chains", "AffectedByHaste"},
		{"Unaffected by Elemental Weakness while affected by Purity of Elements", "CurseEffectOnSelf", "Elemental Weakness", "AffectedByPurityofElements"},
		{"Unaffected by Flammability while affected by Purity of Fire", "CurseEffectOnSelf", "Flammability", "AffectedByPurityofFire"},
		{"Unaffected by Frostbite while affected by Purity of Ice", "CurseEffectOnSelf", "Frostbite", "AffectedByPurityofIce"},
		{"Unaffected by Conductivity while affected by Purity of Lightning", "CurseEffectOnSelf", "Conductivity", "AffectedByPurityofLightning"},
		{"Immune to Ignite while affected by Purity of Fire", "IgniteImmune", "", "AffectedByPurityofFire"},
		{"Immune to Freeze while affected by Purity of Ice", "FreezeImmune", "", "AffectedByPurityofIce"},
		{"Immune to Shock while affected by Purity of Lightning", "ShockImmune", "", "AffectedByPurityofLightning"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			e := ParseMod(tc.line, false)
			if e == nil {
				t.Fatalf("ParseMod(%q) returned nothing", tc.line)
			}
			if e.Extra != "" {
				t.Fatalf("%q left %q unparsed — processModLine discards the whole line", tc.line, e.Extra)
			}
			if len(e.ModList) != 1 {
				t.Fatalf("%q produced %d mods, want 1", tc.line, len(e.ModList))
			}
			m := e.ModList[0]
			if m.Name() != tc.name {
				t.Fatalf("mod name = %q, want %q", m.Name(), tc.name)
			}

			var gotCondition, gotSkillName string
			for _, tag := range m.Tags() {
				switch tt := tag.(type) {
				case *mod.ConditionTag:
					if len(tt.VarList) == 1 {
						gotCondition = tt.VarList[0]
					}
					if tt.Negative {
						t.Errorf("the Condition tag is negated; upstream's is not")
					}
				case *mod.SkillNameTag:
					if len(tt.SkillNameList) == 1 {
						gotSkillName = tt.SkillNameList[0]
					}
				}
			}
			if gotCondition != tc.condition {
				t.Errorf("Condition var = %q, want %q", gotCondition, tc.condition)
			}
			if gotSkillName != tc.skillName {
				t.Errorf("SkillName = %q, want %q", gotSkillName, tc.skillName)
			}
			if tc.name == "CurseEffectOnSelf" {
				if m.Type() != mod.TypeMore {
					t.Errorf("mod type = %v, want MORE", m.Type())
				}
				if v := m.Value().Float(); v != -100 {
					t.Errorf("mod value = %v, want -100", v)
				}
			}
		})
	}

	for _, line := range []string{"Immunity to Ignite", "Immunity to Freeze", "Unaffected by Curses"} {
		if e := ParseMod(line, false); e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Errorf("control %q: ParseMod produced %v, want exactly 1 mod and no Extra", line, e)
		}
	}
}
