package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestNgamahuExtraJewelFuncGrantsAttribute pins Ngamahu.
func TestNgamahuExtraJewelFuncGrantsAttribute(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0282-chieftain-incinerate-of-venting.xml"
	env := corpusEnvForLoadedData(t, build)

	rows := env.Player.ModDB.Tabulate(mod.TypeBase, &moddb.ListCfg{}, "Str")
	granted := 0
	for _, r := range rows {
		if r.Value == 4 {
			granted++
		}
	}

	if granted != 49 {
		t.Errorf("Ngamahu granted %d `Str BASE 4` rows, want 49 (PoB). 0 means the ExtraJewelFunc "+
			"is registered and never executed; a nonzero mismatch means the radius, the node-type "+
			"filter or the non-unique rejection is wrong", granted)
	}
}
