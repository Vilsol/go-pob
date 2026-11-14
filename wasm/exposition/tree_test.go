package exposition

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

// TestCalcTreeVersionIsTheCalculatorsTree: The tree page draws and allocates on CalcTreeVersion, the tree each build is
// calculated on: its active spec's own version, or the latest for a version go-pob does not host.
func TestCalcTreeVersionIsTheCalculatorsTree(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, assetcache.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for recorded, want := range map[pob.TreeVersion]pob.TreeVersion{
		"3_18":               data.TreeVersion3_18,
		"3_28":               data.LatestTreeVersion,
		data.TreeVersion3_29: data.TreeVersion3_29,
		data.TreeVersion3_30: data.TreeVersion3_30,
	} {
		build := builds.EmptyBuild()
		build.Tree.Specs[0].TreeVersion = recorded
		env, _, _, _ := calculator.InitEnv(build, calculator.NewEnvironmentCache(), calculator.OutputModeMain)
		if env.Spec.TreeVersion != CalcTreeVersion(build) {
			t.Errorf("a build recording %s is calculated on %s, but the tree page draws %s", recorded, env.Spec.TreeVersion, CalcTreeVersion(build))
		}
		if CalcTreeVersion(build) != want {
			t.Errorf("a build recording %s draws %s, want %s", recorded, CalcTreeVersion(build), want)
		}
	}
}
