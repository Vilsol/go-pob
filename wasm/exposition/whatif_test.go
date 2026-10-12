package exposition

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestMiscCalculatorStatOutputs: The tree tooltip reads the granted passives and the minion table off Base.
func TestMiscCalculatorStatOutputs(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		file    string
		granted bool
		minion  bool
	}{
		{"0004-trickster-lightning-tendrils-of-eccentricity.xml", true, false},
		{"0008-necromancer-vaal-summon-skeletons.xml", false, true},
	} {
		xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, tc.file))
		if err != nil {
			t.Fatalf("REFUSING: %v", err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		nodes := slices.Clone(build.Build.PassiveNodes)

		misc := GetMiscCalculator(build)
		base := misc.Base()
		if tc.granted && len(base.GrantedPassives) == 0 {
			t.Errorf("%s: no granted passives on Base", tc.file)
		}
		if got := base.Minion != nil; got != tc.minion {
			t.Errorf("%s: minion output present = %v, want %v", tc.file, got, tc.minion)
		}
		if !base.SkillFlags[calculator.SkillFlagAttack] && !base.SkillFlags[calculator.SkillFlagSpell] {
			t.Errorf("%s: base carries neither the attack nor the spell flag", tc.file)
		}

		life := base.Output["Life"]
		misc.CalcNodes(nil, nodes[len(nodes)-1:])
		if !slices.Equal(build.Build.PassiveNodes, nodes) {
			t.Errorf("%s: CalcNodes changed the build's nodes", tc.file)
		}
		if misc.Base().Output["Life"] != life {
			t.Errorf("%s: CalcNodes changed Base", tc.file)
		}
	}
}
