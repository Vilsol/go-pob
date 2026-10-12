package calculator

import (
	"slices"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestGrantsImmunityFlaskLinesParse: ModParser.lua:3345-3350 — the six-rule `grants immunity to <ailment> for (%d+) seconds if used while <state>` block.
func TestGrantsImmunityFlaskLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"Grants Immunity to Ignite for 14 seconds if used while Ignited", "IgniteImmune"},
		{"Grants Immunity to Bleeding for 14 seconds if used while Bleeding", "BleedImmune"},
		{"Grants Immunity to Poison for 14 seconds if used while Poisoned", "PoisonImmune"},
		{"Grants Immunity to Freeze for 14 seconds if used while Frozen", "FreezeImmune"},
		{"Grants Immunity to Chill for 14 seconds if used while Chilled", "ChillImmune"},
		{"Grants Immunity to Shock for 14 seconds if used while Shocked", "ShockImmune"},
	} {
		assertSingleImmuneFlag(t, tc.line, tc.name, "UsingFlask")
	}
}

// TestFlaskEffectImmunityLinesParse: ModParser.lua:3366 and :3368.
func TestFlaskEffectImmunityLinesParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		name string
	}{
		{"Immunity to Poison during Flask Effect", "PoisonImmune"},
		{"Immunity to Poison during Effect", "PoisonImmune"},
		{"Immunity to Shock during Effect", "ShockImmune"},
		{"Immunity to Ignite during Effect", "IgniteImmune"},
		{"Immunity to Curses during Effect", "CurseImmune"},
		{"Immunity to Bleeding during Effect", "BleedImmune"},
		{"Immunety to Poison during Flask Effect", "PoisonImmune"},
		{"Immunity to Poison during flask effect", "PoisonImmune"},
	} {
		assertSingleImmuneFlag(t, tc.line, tc.name, "UsingFlask")
	}

	for _, line := range []string{
		"Immunity to Bleeding and Corrupted Blood during Flask Effect",
		"Immunity to Bleeding and Corrupted Blood during Effect",
	} {
		mods, extra := parseMod(line, 2)
		testza.AssertEqual(t, "", extra, "%q must parse completely", line)
		testza.AssertEqual(t, 2, len(mods), "%q must produce TWO mods", line)
		if len(mods) != 2 {
			continue
		}
		testza.AssertEqual(t, "BleedImmune", mods[0].Name(), "%q", line)
		testza.AssertEqual(t, "CorruptedBloodImmune", mods[1].Name(), "%q", line)
	}
}

func assertSingleImmuneFlag(t *testing.T, line, name, cond string) {
	t.Helper()

	mods, extra := parseMod(line, 1)
	testza.AssertEqual(t, "", extra, "%q must parse completely", line)
	testza.AssertEqual(t, 1, len(mods), "%q must produce exactly one mod", line)
	if len(mods) != 1 {
		return
	}

	m := mods[0]
	testza.AssertEqual(t, name, m.Name(), "%q", line)
	testza.AssertEqual(t, mod.TypeFlag, m.Type(), "%q must be a FLAG, not a BASE avoid mod", line)

	testza.AssertEqual(t, 1, len(m.Tags()), "%q must carry only its condition tag", line)
	for _, tag := range m.Tags() {
		c, ok := tag.(*mod.ConditionTag)
		testza.AssertTrue(t, ok, "%q: tag %T is not a condition", line, tag)
		if ok {
			testza.AssertEqual(t, []string{cond}, c.VarList, "%q", line)
		}
	}
}

// TestMagicRingCritLineParses: ModParser.lua:3975.
func TestMagicRingCritLineParses(t *testing.T) {
	assertSingleAvoidMod(t,
		"Take no Extra Damage from Critical Strikes if you have a Magic Ring in left slot",
		"ReduceCritExtraDamage", "MagicItemInRing 1")
}

// TestStunDurationOnYouParses: ModParser.lua:348 — `["stun duration on you"] = "StunDuration"` in the mod-name list.
func TestStunDurationOnYouParses(t *testing.T) {
	mods, extra := parseMod("50% increased Stun Duration on you", 1)
	testza.AssertEqual(t, "", extra, "the line must parse completely")
	testza.AssertEqual(t, 1, len(mods), "exactly one mod")
	if len(mods) != 1 {
		return
	}
	testza.AssertEqual(t, "StunDuration", mods[0].Name(), "must be the SELF stat, not EnemyStunDuration")
	testza.AssertEqual(t, mod.TypeIncrease, mods[0].Type())
	testza.AssertEqual(t, float64(50), mods[0].Value().Float())
}

func assertSingleAvoidMod(t *testing.T, line, name, cond string) {
	t.Helper()

	mods, extra := parseMod(line, 1)
	testza.AssertEqual(t, "", extra, "%q must parse completely", line)
	testza.AssertEqual(t, 1, len(mods), "%q must produce exactly one mod", line)
	if len(mods) != 1 {
		return
	}

	m := mods[0]
	testza.AssertEqual(t, name, m.Name(), "%q", line)
	testza.AssertEqual(t, mod.TypeBase, m.Type(), "%q", line)
	testza.AssertEqual(t, float64(100), m.Value().Float(), "%q", line)

	testza.AssertEqual(t, 2, len(m.Tags()), "%q must carry its GlobalEffect and its condition", line)
	sawGlobal, sawCond := false, false
	for _, tag := range m.Tags() {
		switch tg := tag.(type) {
		case *mod.GlobalEffectTag:
			sawGlobal = true
			testza.AssertTrue(t, tg.UnscalableTag, "%q: the GlobalEffect tag must be unscalable", line)
			testza.AssertTrue(t, slices.Contains(tg.GlobalEffectList, "Global"), "%q: effectType must be Global", line)
		case *mod.ConditionTag:
			sawCond = true
			testza.AssertEqual(t, []string{cond}, tg.VarList, "%q", line)
		default:
			testza.AssertTrue(t, false, "%q: unexpected tag %T", line, tag)
		}
	}
	testza.AssertTrue(t, sawGlobal, "%q: no GlobalEffect tag", line)
	testza.AssertTrue(t, sawCond, "%q: no condition tag", line)
}
