package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestNgamahuTransformsDamageTypesToFire pins ModParser.lua:2611-2618.
func TestNgamahuTransformsDamageTypesToFire(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "1632-chieftain-incinerate-of-venting-2.xml")

	count, sum := 0, 0.0
	for _, r := range env.Player.ModDB.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "FireDamage") {
		if strings.HasPrefix(string(r.Mod.GetSource()), "Tree:") {
			count++
			sum += r.Value
		}
	}
	if count != 26 || sum != 450 {
		t.Errorf("Tree-sourced FireDamage INC: %d rows summing %v, PoB 26 rows summing 450", count, sum)
	}

	lightning := 0.0
	for _, r := range env.Player.ModDB.Tabulate(mod.TypeIncrease, &moddb.ListCfg{}, "LightningDamage") {
		if strings.HasPrefix(string(r.Mod.GetSource()), "Tree:") {
			lightning += r.Value
		}
	}
	if lightning != 0 {
		t.Errorf("Tree-sourced LightningDamage INC sums %v, PoB 0 (the source is removed)", lightning)
	}

	str := 0
	for _, r := range env.Player.ModDB.Tabulate(mod.TypeBase, &moddb.ListCfg{}, "Str") {
		if r.Value == 4 {
			str++
		}
	}
	if str == 0 {
		t.Errorf("NEIGHBOUR CONTROL: the :2619 sibling produced no `Str BASE 4` rows")
	}
}
