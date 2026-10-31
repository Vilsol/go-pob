package calculator2

import (
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

func configTestEnv(t *testing.T, inputs string) *Environment {
	t.Helper()
	raw, err := os.ReadFile("testdata/from_nothing.xml")
	testza.AssertNoError(t, err)
	xml := strings.Replace(string(raw), `<ConfigSet id="1" title="Default">`, `<ConfigSet id="1" title="Default">`+inputs, 1)
	build, err := pob2.ParseBuild([]byte(xml))
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)
	return env
}

type configModRow struct {
	Type   mod.Type
	Value  float64
	Source mod.Source
}

func configModRows(mods []mod.Mod) []configModRow {
	rows := make([]configModRow, 0, len(mods))
	for _, m := range mods {
		rows = append(rows, configModRow{Type: m.Type(), Value: m.Value().Float(), Source: m.GetSource()})
	}
	return rows
}

// PoB-PoE2's modDB for a build on the default enemy size holds EnemyRadius only as the preset's BASE 3:
// enemySizePreset's SetPlaceholder(3, false) does not reach the placeholder table the enemyRadius
// option applies from.
func TestConfigEnemySizePresetLeavesEnemyRadiusPlaceholder(t *testing.T) {
	env := configTestEnv(t, "")
	testza.AssertEqual(t, []configModRow{{Type: mod.TypeBase, Value: 3, Source: "Config"}}, configModRows(env.ModDB.Mods["EnemyRadius"]))
}

// PoB-PoE2's enemyModDB for a build with Delirious Effect at 100% carries these two mods.
func TestConfigDeliriousPercentageAddsEnemyMods(t *testing.T) {
	env := configTestEnv(t, `<Input name="deliriousPercentage" string="100Percent"/>`)
	var got []configModRow
	for _, name := range []string{"DamageTaken", "Damage"} {
		for _, row := range configModRows(env.Enemy.ModDB.Mods[name]) {
			if row.Source == "100% Delirious" {
				got = append(got, row)
			}
		}
	}
	testza.AssertEqual(t, []configModRow{{Type: mod.TypeMore, Value: -80, Source: "100% Delirious"}, {Type: mod.TypeIncrease, Value: 30, Source: "100% Delirious"}}, got)
}
