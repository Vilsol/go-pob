package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestProjectileAndChainPluralRows pins seven parser rows whose SINGULAR or more general sibling was already here and whose own spelling was not.
//
//pob:block:src/Modules/ModParser.lua:4551-4575
func TestProjectileAndChainPluralRows(t *testing.T) {
	for _, c := range []struct {
		line          string
		name          string
		value         float64
		modFlags      mod.MFlag
		keywordFlags  mod.KeywordFlag
		wantSlotTwo   bool
		wantCondition string
	}{
		{line: "Attacks have an additional Projectile when in Off Hand", name: "ProjectileCount", value: 1, modFlags: mod.MFlagAttack, wantSlotTwo: true},
		{line: "Attacks fire an additional Projectile", name: "ProjectileCount", value: 1, modFlags: mod.MFlagAttack},
		{line: "Spells fire an additional Projectile", name: "ProjectileCount", value: 1, modFlags: mod.MFlagSpell},
		{line: "Skills Chain +1 times", name: "ChainCountMax", value: 1},
		{line: "Projectiles Chain +1 times while you have Phasing", name: "ChainCountMax", value: 1, modFlags: mod.MFlagProjectile, wantCondition: "Phasing"},

		{line: "Attacks fire an additional Projectile when in Off Hand", name: "ProjectileCount", value: 1, modFlags: mod.MFlagAttack, wantSlotTwo: true},
		{line: "Attacks fire 2 additional Projectiles when in Off Hand", name: "ProjectileCount", value: 2, modFlags: mod.MFlagAttack, wantSlotTwo: true},
		{line: "Attacks fire 3 additional Projectiles", name: "ProjectileCount", value: 3, modFlags: mod.MFlagAttack},
		{line: "Spells fire 2 additional Projectiles", name: "ProjectileCount", value: 2, modFlags: mod.MFlagSpell},
		{line: "Projectiles Chain +2 times", name: "ChainCountMax", value: 2, modFlags: mod.MFlagProjectile},
		{line: "Arrows Chain +1 times", name: "ChainCountMax", value: 1, keywordFlags: mod.KeywordFlagArrow},
	} {
		e := ParseMod(c.line, false)
		if e == nil || len(e.ModList) == 0 {
			t.Errorf("no mods produced (line is DROPPED WHOLE): %s", c.line)
			continue
		}
		if extra := strings.TrimSpace(modExtra(e)); extra != "" {
			t.Errorf("leftover %q (line is DROPPED WHOLE): %s", extra, c.line)
			continue
		}
		if len(e.ModList) != 1 {
			t.Errorf("want exactly 1 mod, got %d: %s", len(e.ModList), c.line)
			continue
		}
		m := e.ModList[0]
		if m.Name() != c.name {
			t.Errorf("name %q, want %q: %s", m.Name(), c.name, c.line)
			continue
		}
		if got := m.Value().Float(); got != c.value {
			t.Errorf("value %v, want %v: %s", got, c.value, c.line)
		}
		if m.Flags() != c.modFlags {
			t.Errorf("ModFlags %d, want %d: %s", m.Flags(), c.modFlags, c.line)
		}
		if m.KeywordFlags() != c.keywordFlags {
			t.Errorf("KeywordFlags %d, want %d: %s", m.KeywordFlags(), c.keywordFlags, c.line)
		}
		var slotTwo bool
		var condition string
		for _, tag := range m.Tags() {
			if sn, ok := tag.(*mod.SlotNumberTag); ok && sn.N == 2 {
				slotTwo = true
			}
			if cond, ok := tag.(*mod.ConditionTag); ok && len(cond.VarList) > 0 {
				condition = cond.VarList[0]
			}
		}
		if slotTwo != c.wantSlotTwo {
			t.Errorf("SlotNumber(2) present=%v, want %v: %s", slotTwo, c.wantSlotTwo, c.line)
		}
		if condition != c.wantCondition {
			t.Errorf("condition %q, want %q: %s", condition, c.wantCondition, c.line)
		}
	}
}
