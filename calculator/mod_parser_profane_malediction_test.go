package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestProfaneGroundMaledictionCarriesItsCondition: ModParser.lua:5085, the last absent member of Malediction's six-row family.
func TestProfaneGroundMaledictionCarriesItsCondition(t *testing.T) {
	assertMaledictionRow(t,
		"Profane Ground you create inflicts Malediction on Enemies",
		"OnProfaneGround")

	assertMaledictionRow(t, "Nearby Enemies have Malediction", "")
	assertMaledictionRow(t, "Enemies Blinded by you have Malediction", "Blinded")
}

func assertMaledictionRow(t *testing.T, line, cond string) {
	t.Helper()

	mods, extra := parseMod(line, 1)
	if extra != "" {
		t.Errorf("%q: did not parse completely, extra %q", line, extra)
		return
	}
	if len(mods) != 1 {
		t.Errorf("%q: want exactly one mod, got %d", line, len(mods))
		return
	}

	value := mods[0].Value()
	if value == nil || value.Type() != mod.ModValueMultiTypeList {
		t.Errorf("%q: outer mod is not a LIST", line)
		return
	}
	wrapper, ok := value.ValueList.(mod.EnemyModifier)
	if !ok {
		if ptr, isPtr := value.ValueList.(*mod.EnemyModifier); isPtr {
			wrapper = *ptr
		} else {
			t.Errorf("%q: outer LIST payload is %T, not an EnemyModifier -- a row parsed into the "+
				"wrong ACTOR satisfies every name-based census and is never read", line, value.ValueList)
			return
		}
	}

	inner := wrapper.Mod
	if inner == nil {
		t.Errorf("%q: EnemyModifier wraps no mod", line)
		return
	}
	if inner.Name() != "HasMalediction" {
		t.Errorf("%q: nested mod is %q, want HasMalediction", line, inner.Name())
		return
	}

	var gotConds []string
	for _, raw := range inner.Tags() {
		c, isCond := raw.(*mod.ConditionTag)
		if !isCond {
			t.Errorf("%q: nested mod carries a %T; upstream carries only a Condition", line, raw)
			continue
		}
		gotConds = append(gotConds, c.VarList...)
	}

	if cond == "" {
		if len(gotConds) != 0 {
			t.Errorf("%q: nested flag carries conditions %v and the pin's row carries NONE -- a "+
				"neighbouring key has been widened until it swallowed this line", line, gotConds)
		}
		return
	}
	if len(gotConds) != 1 || gotConds[0] != cond {
		t.Errorf("%q: nested flag conditions are %v, want exactly [%s]. Without it Malediction's "+
			"untyped `Damage INC -10` applies to every enemy of every carrier, not only on that "+
			"ground -- over-application, which no missing-thing instrument here reports", line, gotConds, cond)
	}
}
