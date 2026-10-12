package calculator

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

var shockAvoidAppliesLines = []string{
	"chance to avoid being shocked applies to all elemental ailments",
	"modifiers to chance to avoid being shocked apply to all elemental ailments",
}

// TestShockAvoidAppliesToElementalAilmentsIsParsed pins the producer for a flag this tree already had TWO consumers for and no producer at all.
func TestShockAvoidAppliesToElementalAilmentsIsParsed(t *testing.T) {
	t.Parallel()

	for _, line := range shockAvoidAppliesLines {
		entry := ParseMod(line, false)
		if entry.ModList == nil {
			t.Fatalf("%q did not parse at all", line)
		}
		found := false
		for _, m := range entry.ModList {
			if m.Name() == "ShockAvoidAppliesToElementalAilments" {
				found = true
			}
		}
		if !found {
			t.Errorf("%q produced no ShockAvoidAppliesToElementalAilments flag; got %d mod(s)",
				line, len(entry.ModList))
		}
	}
}

// TestShockAvoidAppliesToElementalAilmentsMovesTheCorpus grades the elemental AvoidChance keys on the builds that actually carry the line, against PoB.
func TestShockAvoidAppliesToElementalAilmentsMovesTheCorpus(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var active, seen, wrong, shockSeen, shockWrong, mechSeen int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if !env.Player.ModDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
			continue
		}
		active++
		avoidShock := env.Player.ModDB.Sum(mod.TypeBase, nil, "AvoidShock")
		inputAgrees := oracleClose(env.Player.Output["ShockAvoidChance"],
			rec.Output["ShockAvoidChance"], oracleDumpTolerance)
		for _, ailment := range data.ElementalAilment("").Values() {
			key := string(ailment + "AvoidChance")
			want, has := rec.Output[key]
			if !has {
				continue
			}
			got := env.Player.Output[key]
			if env.Player.ModDB.Flag(nil, string(ailment+"Immune"), "ElementalAilmentImmune") {
				continue
			}
			base := env.Player.ModDB.Sum(mod.TypeBase, nil,
				string("Avoid"+ailment), "AvoidAilments", "AvoidElementalAilments")
			without := math.Floor(math.Min(base, 100))
			with := math.Floor(math.Min(base+avoidShock, 100))

			if ailment == data.AilmentShock {
				if without == with {
					continue
				}
				shockSeen++
				if got != without {
					shockWrong++
				}
				continue
			}
			if without == with {
				continue
			}
			mechSeen++
			if got != with {
				t.Errorf("%s %s: published %v, expected the with-term form %v (without %v)",
					name, key, got, with, without)
			}
			if !inputAgrees {
				continue
			}
			seen++
			if !oracleClose(got, want, oracleDumpTolerance) {
				wrong++
				t.Logf("%s %s: ours %v, PoB %v (without term %v, with term %v)",
					name, key, got, want, without, with)
			}
		}
		if active >= 100 {
			break
		}
	}

	if active == 0 {
		t.Fatalf("REFUSING: the flag is live on NO graded build, so this test measures nothing. "+
			"The line is present in the input XML of 373 builds; either the parser entry (%q) "+
			"regressed or the mod is no longer reaching the player modDB",
			strings.Join(shockAvoidAppliesLines, " / "))
	}
	if shockSeen == 0 {
		t.Fatal("REFUSING: no ShockAvoidChance is published on the active builds, so the control " +
			"that the term must NOT apply to Shock itself is absent")
	}
	if mechSeen == 0 {
		t.Fatalf("REFUSING: over %d builds with the flag live, the term never changes a value, "+
			"so nothing here grades it", active)
	}
	if seen == 0 {
		t.Fatal("REFUSING: no decisive pair survives the input-agreement filter, so the oracle " +
			"half of this test measured nothing")
	}
	if wrong > 0 {
		t.Errorf("non-Shock elemental AvoidChance: %d of %d pairs disagree with PoB on the %d "+
			"builds where the flag is live and ShockAvoidChance already agrees", wrong, seen, active)
	}
	if shockWrong > 0 {
		t.Errorf("CONTROL BROKEN: ShockAvoidChance disagrees on %d of %d pairs — the term is "+
			"being applied to Shock itself, which PoB's :1584 local excludes", shockWrong, shockSeen)
	}
}
