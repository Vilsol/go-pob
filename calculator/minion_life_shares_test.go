package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestMinionLifeSharesFoldsTheSpecificPool pins CalcPerform.lua:3625-3629 against its consumer at CalcDefence.lua:2476-2491.
func TestMinionLifeSharesFoldsTheSpecificPool(t *testing.T) {
	requireGameData(t, "3.29")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("needs the corpus")
	}

	const build = "4374-ladder-CongTuBungTo_5987.laogialomo"
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatal(err)
	}
	pob, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)
	out := env.Player.Output

	if !env.ModDB.Flag(nil, "MinionLifeSharesTotalStoneGolemLife") {
		t.Fatal("MinionLifeSharesTotalStoneGolemLife is not set -- CalcPerform.lua:3628 did not fire")
	}

	for _, tc := range []struct {
		key  string
		want float64
	}{
		{"MinionAllyDamageMitigation", 18.75},
		{"StoneGolemAllyDamageMitigation", 0},
		{"TotalMinionLife", 7151},
	} {
		if got := out[tc.key]; math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s = %v, want %v", tc.key, got, tc.want)
		}
	}

	if _, ok := out["TotalStoneGolemLife"]; ok {
		t.Errorf("TotalStoneGolemLife = %v is still published; PoB deletes the key", out["TotalStoneGolemLife"])
	}

	for _, tc := range []struct {
		key  string
		want float64
	}{
		{"FireTotalHitPool", 6711.2205128205},
		{"FireMaximumHitTaken", 6373},
	} {
		if got := out[tc.key]; math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("%s = %v, want %v", tc.key, got, tc.want)
		}
	}
}
