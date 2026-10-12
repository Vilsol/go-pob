package calculator

import (
	"math"
	"os"
	"testing"
)

var wildSavageryBuilds = []string{
	"0235-champion-sunder.xml",
	"0733-champion-static-strike-of-gathering-lightning.xml",
	"1247-slayer-cyclone-of-tumult-14.xml",
	"1499-slayer-sunder-of-earthbreaking.xml",
	"1505-slayer-sunder-of-earthbreaking-2.xml",
	"1532-champion-lacerate-of-butchering.xml",
	"1546-slayer-sunder-of-earthbreaking-4.xml",
}

func TestWildSavageryReachesActionSpeed(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range wildSavageryBuilds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, present := rec.Output["ActionSpeedMod"]
		if !present {
			t.Fatalf("REFUSING: PoB reports no ActionSpeedMod for %s", build)
		}

		env := corpusEnvForLoadedData(t, build)

		if !env.ModDB.Flag(nil, "Condition:WildSavagery") {
			t.Errorf("%s: Condition:WildSavagery is not set — ConfigOptions.lua:1065-1067's "+
				"buffWildSavagery handler (or its configOrder entry) is missing", build)
		}
		if !env.ModDB.Flag(nil, "WildSavagery") {
			t.Errorf("%s: the WildSavagery flag is not set — ModParser.lua:3080's tree row is "+
				"missing", build)
		}

		if got := env.Player.Output["ActionSpeedMod"]; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: ActionSpeedMod = %v, PoB %v (difference %v; Wild Savagery is "+
				"ActionSpeed INC 10)", build, got, want, want-got)
		}
		if got, w := env.Player.Output["StunAvoidChance"], rec.Output["StunAvoidChance"]; w != 0 && got != w {
			t.Errorf("%s: StunAvoidChance = %v, PoB %v — the same block's StunImmune flag",
				build, got, w)
		}
	}
}
