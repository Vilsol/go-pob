package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestTheIronMassArmFeedsItsSkeletons pins CalcSetup.lua:1051-1068, the one arm of the equipped-item merge chain that had no port.
func TestTheIronMassArmFeedsItsSkeletons(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	build := loadCorpusBuild(t, "0008-necromancer-vaal-summon-skeletons.xml")
	env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

	if env.TheIronMass == nil {
		t.Fatalf("env.TheIronMass is nil; CalcSetup.lua:1053 builds it for a Weapon 1 named " +
			"\"The Iron Mass, Gladius\", and this build equips one")
	}
	if n := len(env.TheIronMass.Mods()); n == 0 {
		t.Errorf("env.TheIronMass carries 0 mods; the arm at CalcSetup.lua:1054-1065 adds every "+
			"mod without a SocketedIn tag, and the item has %d", n)
	}
}
