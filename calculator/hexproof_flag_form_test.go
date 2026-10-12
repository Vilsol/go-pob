package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestHexproofFlagFormBuildsTheDescribedMod: ModParser.lua:4137-4141 — a FLAG form whose flagTypes entry is a TABLE does not build `<name> FLAG true`.
func TestHexproofFlagFormBuildsTheDescribedMod(t *testing.T) {
	for _, tc := range []struct {
		line string
		cond string
	}{
		{"You are Hexproof if you have a Magic Ring in right slot", "MagicItemInRing 2"},
		{"You are Hexproof if you have a Magic Ring in left slot", "MagicItemInRing 1"},
	} {
		mods, extra := parseMod(tc.line, 1)
		testza.AssertEqual(t, "", extra, "%q must parse completely", tc.line)
		testza.AssertEqual(t, 1, len(mods), "%q must produce exactly one mod", tc.line)
		if len(mods) != 1 {
			continue
		}

		m := mods[0]
		testza.AssertEqual(t, "CurseEffectOnSelf", m.Name(), "%q", tc.line)
		testza.AssertEqual(t, mod.TypeMore, m.Type(), "%q must be MORE, not BASE", tc.line)
		testza.AssertEqual(t, float64(-100), m.Value().Float(), "%q", tc.line)

		testza.AssertEqual(t, 1, len(m.Tags()), "%q must carry only its condition tag", tc.line)
		for _, tag := range m.Tags() {
			cond, isCond := tag.(*mod.ConditionTag)
			testza.AssertTrue(t, isCond, "%q: tag %T is not a condition", tc.line, tag)
			if isCond {
				testza.AssertEqual(t, []string{tc.cond}, cond.VarList, "%q", tc.line)
			}
		}
	}
}

// TestPlainFlagFormIsUnchanged: The ordinary FLAG entries must keep building `<name> FLAG true` — the table form is the exception, not the rule.
func TestPlainFlagFormIsUnchanged(t *testing.T) {
	mods, extra := parseMod("You have Onslaught", 1)
	testza.AssertEqual(t, "", extra)
	testza.AssertEqual(t, 1, len(mods))
	if len(mods) != 1 {
		return
	}
	testza.AssertEqual(t, "Condition:Onslaught", mods[0].Name())
	testza.AssertEqual(t, mod.TypeFlag, mods[0].Type())
	testza.AssertTrue(t, mods[0].Value().Flag(), "a plain FLAG entry carries the boolean true")
}
