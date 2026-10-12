package exposition

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

// TestCalcTreeVersionIsTheCalculatorsTree: The tree page draws and allocates on CalcTreeVersion.
func TestCalcTreeVersionIsTheCalculatorsTree(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, recorded := range []string{"3_18", "3_28", "3_29"} {
		build := builds.EmptyBuild()
		build.Tree.Specs[0].TreeVersion = pob.TreeVersion(recorded)
		env, _, _, _ := calculator.InitEnv(build, calculator.NewEnvironmentCache(), calculator.OutputModeMain)
		if env.Spec.TreeVersion != CalcTreeVersion() {
			t.Errorf("a build recording %s is calculated on %s, but the tree page draws %s", recorded, env.Spec.TreeVersion, CalcTreeVersion())
		}
	}
}
