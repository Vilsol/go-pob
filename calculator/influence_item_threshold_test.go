package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestInfluenceItemThresholdRowsParse pins the "if N <influence> Items are Equipped" family.
func TestInfluenceItemThresholdRowsParse(t *testing.T) {
	for _, tc := range []struct {
		line      string
		name      string
		threshold float64
		variable  string
	}{
		{"Cannot be Frozen if 6 Redeemer Items are Equipped", "FreezeImmune", 6, "RedeemerItem"},
		{"Cannot be Stunned if 6 Elder Items are Equipped", "StunImmune", 6, "ElderItem"},
		{"Cannot take Reflected Elemental Damage if 4 Shaper Items are Equipped", "ElementalReflectedDamageTaken", 4, "ShaperItem"},
		{"You can apply an additional Curse if 6 Hunter Items are Equipped", "EnemyCurseLimit", 6, "HunterItem"},
		{"Skills fire an additional Projectile if 6 Crusader Items are Equipped", "ProjectileCount", 6, "CrusaderItem"},
		{"Consecrated Ground around you while stationary if 6 Warlord Items are Equipped", "Condition:OnConsecratedGround", 6, "WarlordItem"},
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
			if len(m.Tags()) != 1 {
				t.Fatalf("got %d tags, want exactly 1 MultiplierThreshold", len(m.Tags()))
			}
			mt, ok := m.Tags()[0].(*mod.MultiplierThresholdTag)
			if !ok {
				t.Fatalf("tag is %#v, want a MultiplierThreshold", m.Tags()[0])
			}
			if mt.Variable != tc.variable {
				t.Errorf("threshold var = %q, want %q", mt.Variable, tc.variable)
			}
			if mt.TagThreshold == nil || *mt.TagThreshold != tc.threshold {
				t.Errorf("threshold = %v, want %v", mt.TagThreshold, tc.threshold)
			}
		})
	}

	for _, line := range []string{"Cannot be Frozen", "Cannot be Stunned", "Cannot take Reflected Elemental Damage"} {
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) != 1 || e.Extra != "" {
			t.Errorf("control %q: ParseMod produced %v, want exactly 1 mod and no Extra", line, e)
			continue
		}
		for _, tg := range e.ModList[0].Tags() {
			if _, isMT := tg.(*mod.MultiplierThresholdTag); isMT {
				t.Errorf("control %q picked up a MultiplierThreshold tag; the unconditional row must stay unconditional", line)
			}
		}
	}
}
