package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestEnemyConfigModsSourcedEnemyConfig: ConfigOptions.lua:2219-2236 source the enemy's resistance and reduction mods "EnemyConfig".
func TestEnemyConfigModsSourcedEnemyConfig(t *testing.T) {
	for _, name := range []string{"enemyPhysicalReduction", "enemyLightningResist", "enemyColdResist", "enemyFireResist", "enemyChaosResist", "enemyMaxResist"} {
		modList, enemyModList := moddb.NewModList(), moddb.NewModList()
		var val interface{} = float64(30)
		if name == "enemyMaxResist" {
			val = true
		}
		configurations[name](val, modList, enemyModList)
		mods := enemyModList.Mods()
		if len(mods) != 1 {
			t.Fatalf("%s: %d enemy mods", name, len(mods))
		}
		if src := string(mods[0].GetSource()); src != "EnemyConfig" {
			t.Errorf("%s: source %q, PoB \"EnemyConfig\"", name, src)
		}
	}
}
