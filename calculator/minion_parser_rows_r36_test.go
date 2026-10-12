package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func parseOneMinionMod(t *testing.T, line string) (mod.Mod, mod.Mod) {
	t.Helper()
	e := ParseMod(strings.ToLower(line), false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("%q: got %+v, want one mod parsed whole", line, e)
	}
	outer := e.ModList[0]
	if outer.Name() != "MinionModifier" || outer.Type() != mod.TypeList {
		t.Fatalf("%q: got %s %s, want MinionModifier LIST", line, outer.Name(), outer.Type())
	}
	mm, ok := outer.Value().List().(mod.MinionModifier)
	if !ok {
		t.Fatalf("%q: value %T, want mod.MinionModifier", line, outer.Value().List())
	}
	return outer, mm.Mod
}

func conditionVars(m mod.Mod) []string {
	var out []string
	for _, tag := range m.Tags() {
		if c, ok := tag.(*mod.ConditionTag); ok {
			out = append(out, c.VarList...)
		}
	}
	return out
}

// TestPerAnimatedWeaponIsParentPerStat pins ModParser.lua:1551.
func TestPerAnimatedWeaponIsParentPerStat(t *testing.T) {
	for _, tc := range []struct{ line, stat string }{
		{"Animated Guardian deals 5% increased Damage per Animated Weapon", "ActiveAnimatedWeaponLimit"},
		{"2% increased Minion Attack and Cast Speed per Skeleton you own", "ActiveSkeletonLimit"},
	} {
		_, inner := parseOneMinionMod(t, tc.line)
		found := false
		for _, tag := range inner.Tags() {
			if ps, ok := tag.(*mod.PerStatTag); ok && slices.Contains(ps.StatList, tc.stat) && ps.TagActor == "parent" {
				found = true
			}
			if _, ok := tag.(*mod.MultiplierTag); ok {
				t.Errorf("%q: carries a Multiplier tag; upstream is PerStat only", tc.line)
			}
		}
		if !found {
			t.Errorf("%q: no PerStat{%s, actor=parent} on %s", tc.line, tc.stat, inner.Name())
		}
	}
}

// TestMinionPhysicalDamageReductionRows pins ModParser.lua:5638-5643.
func TestMinionPhysicalDamageReductionRows(t *testing.T) {
	_, inner := parseOneMinionMod(t, "Minions Hits have 50% chance to ignore Enemy Physical Damage Reduction")
	if inner.Name() != "ChanceToIgnoreEnemyPhysicalDamageReduction" || inner.Type() != mod.TypeBase || inner.Value().Float() != 50 {
		t.Errorf("ignore row: got %s %s %v", inner.Name(), inner.Type(), inner.Value())
	}

	_, inner = parseOneMinionMod(t, "Minions Attacks Overwhelm 10% Physical Damage Reduction")
	if inner.Name() != "EnemyPhysicalDamageReduction" || inner.Value().Float() != -10 {
		t.Errorf("overwhelm row: got %s %v", inner.Name(), inner.Value())
	}
	hasAttack := false
	for _, tag := range inner.Tags() {
		if st, ok := tag.(*mod.SkillTypeTag); ok && slices.Contains(st.SkillTypeList, "Attack") && !st.Negative {
			hasAttack = true
		}
	}
	if !hasAttack {
		t.Error("overwhelm row: missing SkillType Attack tag (ModParser.lua:5639)")
	}

	e := ParseMod(strings.ToLower("Hits have 50% chance to ignore Enemy Physical Damage Reduction"), false)
	if e == nil || e.Extra != "" || len(e.ModList) != 1 || e.ModList[0].Name() != "ChanceToIgnoreEnemyPhysicalDamageReduction" {
		t.Errorf("player control: got %+v", e)
	}
}

// TestGhastlyEyeUnholyMightRow pins ModParser.lua:5435-5437 (Command of the Pit, Data/Uniques/ gloves.lua:703).
func TestGhastlyEyeUnholyMightRow(t *testing.T) {
	outer, inner := parseOneMinionMod(t, "With a Ghastly Eye Jewel Socketed, Minions have 25% chance to gain Unholy Might on Hit with Spells")
	if !slices.Contains(conditionVars(outer), "HaveGhastlyEyeJewelIn{SlotName}") {
		t.Errorf("outer conditions %v, want HaveGhastlyEyeJewelIn{SlotName}", conditionVars(outer))
	}
	if inner.Name() != "Condition:UnholyMight" || inner.Type() != mod.TypeFlag {
		t.Errorf("inner: got %s %s", inner.Name(), inner.Type())
	}
	if !slices.Contains(conditionVars(inner), "HitSpellRecently") {
		t.Errorf("inner conditions %v, want HitSpellRecently on the MINION mod", conditionVars(inner))
	}
	if slices.Contains(conditionVars(outer), "HitSpellRecently") {
		t.Error("HitSpellRecently is on the outer (player) mod; upstream puts it on the minion's")
	}

	outer, inner = parseOneMinionMod(t, "With a Ghastly Eye Jewel Socketed, Minions have +1000 to Accuracy Rating")
	if !slices.Contains(conditionVars(outer), "HaveGhastlyEyeJewelIn{SlotName}") || inner.Name() != "Accuracy" {
		t.Errorf("accuracy control: outer %v inner %s", conditionVars(outer), inner.Name())
	}
}
