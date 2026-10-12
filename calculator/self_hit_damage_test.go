package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSelfHitDamageSourceRowsParse pins the seven "Self hit dmg" rows of ModParser.lua:3252-3273.
func TestSelfHitDamageSourceRowsParse(t *testing.T) {
	cases := []struct {
		line    string
		name    string
		baseDmg float64
		dmgMult float64
		dmgType string
	}{
		{"Take 245 Fire Damage when you Use a Skill", "EnmitysEmbraceSelfDamage", 245, 0, "Fire"},
		{"Take 100 Fire Damage when you Ignite an Enemy", "EyeOfInnocenceSelfDamage", 100, 0, "Fire"},
		{"350 Physical Damage taken on Minion Death", "HeartboundLoopSelfDamage", 350, 0, "Physical"},
		{"Take 250 Lightning Damage when Herald of Thunder Hits an Enemy", "StormSecretSelfDamage", 250, 0, "Lightning"},
		{"Your Skills deal you 400% of Mana Cost as Physical Damage", "ScoldsBridleSelfDamage", 0, 400, "Physical"},
		{"Your Skills deal you 300% of Mana spent on Upfront Skill Mana Costs as Physical Damage", "ScoldsBridleSelfDamage", 0, 300, "Physical"},
		{"When you Attack, take 25% of Life as Physical Damage for each Warcry Exerting the Attack", "EchoesOfCreationSelfDamage", 0, 25, "Physical"},
	}
	for _, c := range cases {
		entry := ParseMod(c.line, false)
		if strings.Trim(entry.Extra, " ") != "" {
			t.Errorf("%q: unparsed remainder %q — an unparsed remainder discards the WHOLE line", c.line, entry.Extra)
			continue
		}
		if len(entry.ModList) != 1 {
			t.Errorf("%q: got %d mods, want 1", c.line, len(entry.ModList))
			continue
		}
		m := entry.ModList[0]
		if m.Name() != c.name {
			t.Errorf("%q: mod name %q, want %q", c.line, m.Name(), c.name)
			continue
		}
		v := m.Value()
		sd, ok := v.List().(mod.SelfDamage)
		if !ok {
			t.Errorf("%q: payload is %T, want mod.SelfDamage", c.line, v.List())
			continue
		}
		if sd.BaseDamage != c.baseDmg || sd.DmgMult != c.dmgMult || sd.DamageType != c.dmgType {
			t.Errorf("%q: payload %+v, want {BaseDamage:%v DmgMult:%v DamageType:%q}",
				c.line, sd, c.baseDmg, c.dmgMult, c.dmgType)
		}
	}
}

// TestSelfHitDamageTypeIsTitleCasedLikeTheGsub pins CalcOffence.lua's `string.gsub(" "..value.damageType, "%W%l", string.upper):sub(2)`.
func TestSelfHitDamageTypeIsTitleCasedLikeTheGsub(t *testing.T) {
	for in, want := range map[string]string{
		"fire":       "Fire",
		"physical":   "Physical",
		"cold storm": "Cold Storm",
		"a-b":        "A-B",
		"Fire":       "Fire",
	} {
		if got := selfHitDamageTypeName(in); got != want {
			t.Errorf("selfHitDamageTypeName(%q) = %q, want %q", in, got, want)
		}
	}
}
