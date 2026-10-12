package calculator

import (
	"os"
	"testing"
)

var sacrificeMinionArrowBuilds = []string{
	"0984-deadeye-tornado-shot.xml",
	"7124-ladder-dunadan8_0909.ElisabethCygne.xml",
	"7656-ladder-lmaoto_3902.ALLFLAME_DL.xml",
	"9848-ladder-Enzeq_0448.RiseVest.xml",
}

func TestSacrificeMinionGrantsAdditionalArrows(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range sacrificeMinionArrowBuilds {
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		want, present := rec.Output["ProjectileCount"]
		if !present || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero ProjectileCount for %s", build)
		}

		env := corpusEnvForLoadedData(t, build)

		if !env.ModDB.Flag(nil, "Condition:HaveDamageableMinion") {
			t.Errorf("%s: Condition:HaveDamageableMinion is not set — CalcPerform.lua:1426-1428's "+
				"producer is missing", build)
		}
		if !env.ModDB.Flag(nil, "Condition:SacrificeMinionOnAttack") {
			t.Errorf("%s: Condition:SacrificeMinionOnAttack is not set — configs.go's "+
				"conditionSacrificeMinion handler defaults to TRUE and should have fired", build)
		}

		if got := env.Player.Output["ProjectileCount"]; got != want {
			t.Errorf("%s: ProjectileCount = %v, PoB %v — ModParser.lua:4579-4581 adds the "+
				"sacrificed minion's arrows", build, got, want)
		}
	}
}
