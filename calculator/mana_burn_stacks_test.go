package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

// TestManaBurnStackMultipliers pins CalcPerform.lua:902-915 — the mana-burn stack block of doActorMisc.
func TestManaBurnStackMultipliers(t *testing.T) {
	requireGameData(t, "3.29")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("needs the corpus")
	}

	for _, build := range []string{
		"1064-guardian-dominating-blow-of-inspiring-4",
		"1411-guardian-dominating-blow-of-inspiring-6",
		"0625-elementalist-winter-orb-34",
		"1539-elementalist-corrupting-fever-13",
	} {
		t.Run(build, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
			if err != nil {
				t.Fatal(err)
			}
			pob, err := builds.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			if got := pob.GetNumberOption("multiplierManaBurnStacks"); got != 1 {
				t.Fatalf("REFUSING: multiplierManaBurnStacks reads %v, want 1", got)
			}
			env := (&Calculator{PoB: pob}).BuildOutput(OutputModeMain)
			for _, tc := range []struct {
				name string
				want float64
			}{
				{"Multiplier:ManaBurnStacks", 1},
				{"Multiplier:EffectiveManaBurnStacks", 1},
				{"Multiplier:WeepingWoundsStacks", 0},
			} {
				if got := env.ModDB.Sum(mod.TypeBase, nil, tc.name); got != tc.want {
					t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
				}
			}
		})
	}
}
