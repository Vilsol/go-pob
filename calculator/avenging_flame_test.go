package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestSocketedProjectileSpellCooldownParse pins ModParser.lua:3383-3385, Black Zenith's line (ModCache.lua:12093).
func TestSocketedProjectileSpellCooldownParse(t *testing.T) {
	mods, extra := parseMod("Socketed Projectile Spells have +4 seconds to Cooldown", 1)
	testza.AssertEqual(t, "", extra, "the line must parse completely")
	if len(mods) != 1 {
		t.Fatalf("produced %d mods, want 1", len(mods))
	}
	m := mods[0]
	testza.AssertEqual(t, "CooldownRecovery", m.Name())
	testza.AssertEqual(t, mod.TypeBase, m.Type())
	testza.AssertEqual(t, 4.0, m.Value().ValueFloat)
	if len(m.Tags()) != 3 {
		t.Fatalf("carries %d tags, want SocketedIn + SkillType Projectile + SkillType Spell", len(m.Tags()))
	}
	if s, ok := m.Tags()[0].(*mod.SocketedInTag); !ok || s.SlotName != "{SlotName}" {
		t.Errorf("tag 0 = %#v, want SocketedIn {SlotName}", m.Tags()[0])
	}
	want := []data.SkillType{data.SkillTypeProjectile, data.SkillTypeSpell}
	for i, w := range want {
		st, ok := m.Tags()[i+1].(*mod.SkillTypeTag)
		if !ok || len(st.SkillTypeList) != 1 || st.SkillTypeList[0] != string(w) {
			t.Errorf("tag %d = %#v, want SkillType %s", i+1, m.Tags()[i+1], w)
		}
	}

	control, extra := parseMod("Socketed Projectile Spells have 80% less Skill Effect Duration", 1)
	testza.AssertEqual(t, "", extra, "the neighbour line must still parse completely")
	if len(control) != 1 || control[0].Name() != "ExtraSkillMod" {
		t.Fatalf("neighbour produced %v, want one ExtraSkillMod", control)
	}
}

// TestAvengingFlameTriggerAndDamage pins CalcTriggers.lua:1475-1482 and sup_str.lua:2539-2547 on the three corpus builds that main-skill Flamewood's Avenging Flame.
func TestAvengingFlameTriggerAndDamage(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := []struct {
		build               string
		speed, fireMinBase  float64
		effectiveSourceRate float64
	}{
		{"1037-hierophant-rejuvenation-totem.xml", 0.199362, 18396, 0.199362},
		{"1190-hierophant-rejuvenation-totem-2.xml", 0.199362, 39840.48, 0.199362},
	}
	for _, c := range cases {
		env := corpusEnvForLoadedData(t, c.build)
		out := env.Player.Output
		if got := out["Speed"]; math.Abs(got-c.speed)/c.speed > 1e-5 {
			t.Errorf("%s: Speed = %v, PoB %v", c.build, got, c.speed)
		}
		if got := out["EffectiveSourceRate"]; math.Abs(got-c.effectiveSourceRate)/c.effectiveSourceRate > 1e-5 {
			t.Errorf("%s: EffectiveSourceRate = %v, PoB %v", c.build, got, c.effectiveSourceRate)
		}
		if got := out["FireMinBase"]; math.Abs(got-c.fireMinBase) > 0.5 {
			t.Errorf("%s: FireMinBase = %v, PoB %v", c.build, got, c.fireMinBase)
		}
	}
}
