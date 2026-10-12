package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestDiagReportModNilCfgDoesNotPanic: `task diag STEP=tags ACTOR=enemy CFG=nil` panicked in diagReportMod's EXCL-CFG explanation.
func TestDiagReportModNilCfgDoesNotPanic(t *testing.T) {
	store := moddb.NewModDB()
	flagged := mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 10).Flag(mod.MFlagAttack)
	keyworded := mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 10).KeywordFlag(mod.KeywordFlagAttack)
	for _, m := range []mod.Mod{flagged, keyworded} {
		if a, b, c := moddb.CfgAdmits(m, nil); a && b && c {
			t.Fatalf("fixture does not reach the EXCL-CFG branch: %v admitted under nil cfg", m)
		}
		if got := diagReportMod(store, m, nil, 0, "db"); got != 0 {
			t.Fatalf("excluded mod contributed %v", got)
		}
	}
}
