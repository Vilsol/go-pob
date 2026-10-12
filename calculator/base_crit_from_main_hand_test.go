package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestBaseCritFromMainHandParses covers ModParser.lua:5477-5478.
func TestBaseCritFromMainHandParses(t *testing.T) {
	line := "Base Spell Critical Strike Chance of Spells is equal to that of Main Hand Weapon"
	mods, extra := parseMod(line, 1)
	testza.AssertEqual(t, "", extra, "%q must parse completely", line)
	testza.AssertEqual(t, 1, len(mods), "%q must produce exactly one mod", line)
	if len(mods) != 1 {
		return
	}
	testza.AssertEqual(t, "BaseCritFromMainHand", mods[0].Name(), "%q", line)
	testza.AssertEqual(t, mod.MFlagSpell, mods[0].Flags(), "%q must carry ModFlag.Spell", line)
}

// TestBaseCritFromMainHandReplacesBaseCrit covers CalcOffence.lua:2998-3004.
func TestBaseCritFromMainHandReplacesBaseCrit(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	builds := []string{
		"0233-elementalist-frostblink-of-wintry-blast.xml",
		"0421-hierophant-vaal-arc.xml",
		"0442-inquisitor-purifying-flame-of-revelations.xml",
		"0452-inquisitor-vaal-flameblast.xml",
		"0739-elementalist-frostblink-of-wintry-blast-2.xml",
		"0804-inquisitor-firestorm-of-meteors.xml",
		"0827-elementalist-crackling-lance.xml",
		"1402-trickster-fireball.xml",
		"1557-hierophant-flame-surge.xml",
		"1564-hierophant-fireball-2.xml",
		"1585-hierophant-fireball-3.xml",
		"1976-ladder-gdm12322_5451.Yethello_CotA.xml",
		"2082-ladder-ALKEROX_2244.ALKEROX.xml",
	}

	var seen, agreed, flagged int
	for _, build := range builds {
		rec, ok := oracle[build]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want := rec.Output["PreEffectiveCritChance"]
		if want <= 0 {
			continue
		}
		seen++
		env := corpusEnvForLoadedData(t, build)

		if env.Player.MainSkill != nil &&
			env.Player.MainSkill.SkillModList.Flag(env.Player.MainSkill.SkillCfg, "BaseCritFromMainHand") {
			flagged++
		}
		if oracleClose(env.Player.Output["PreEffectiveCritChance"], want, oracleDumpTolerance) {
			agreed++
		}
	}

	if seen == 0 {
		t.Fatal("REFUSING: none of the BaseCritFromMainHand population is graded; this test no " +
			"longer measures the mechanism it was written for")
	}
	t.Logf("BaseCritFromMainHand: %d graded, flag resolves on %d, PreEffectiveCritChance agrees on %d",
		seen, flagged, agreed)

	const flagResolveFloor = 12
	if flagged < flagResolveFloor {
		t.Errorf("the flag resolves under the skill cfg on only %d of %d builds PoB records it on; "+
			"the producer is not reaching the consumer", flagged, seen)
	}

	const agreementFloor = 9
	if agreed < agreementFloor {
		t.Errorf("PreEffectiveCritChance agrees with PoB on %d of %d BaseCritFromMainHand builds, "+
			"below the floor of %d measured when the substitution landed", agreed, seen, agreementFloor)
	}
}
