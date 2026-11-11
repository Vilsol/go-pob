package exposition

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

// configWhatIfOracle is oracle/poe2_config_whatif.lua over sorceress_spark.xml (`mise run config-whatif-poe2`).
type configWhatIfOracle struct {
	Options []struct {
		Var   string
		Value any
		Lines []string
	}
	Custom []struct {
		Group int
		Lines []string
	}
	Placeholders []struct {
		Inputs       string
		Placeholders map[string]float64
	}
}

func readConfigWhatIfOracle(t *testing.T) configWhatIfOracle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_config_whatif_sorceress_spark.json"))
	mustOK(t, err)
	var o configWhatIfOracle
	mustOK(t, json.Unmarshal(raw, &o))
	return o
}

// Every option control's placeholder PoB-PoE2 shows after loading the fixture with each recorded input set.
func TestConfigPlaceholdersMatchPoB(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("testdata", "sorceress_spark.xml"))
	mustOK(t, err)
	o := readConfigWhatIfOracle(t)
	if len(o.Placeholders) < 6 {
		t.Fatalf("%d placeholder sets recorded, want 6", len(o.Placeholders))
	}
	for _, set := range o.Placeholders {
		calc, err := ParseBuild(strings.Replace(string(xml), `<ConfigSet id="1" title="Default">`, `<ConfigSet id="1" title="Default">`+set.Inputs, 1))
		mustOK(t, err)
		recalc(t, calc)
		got := calc.ConfigPlaceholders()
		if !maps.Equal(got, set.Placeholders) {
			for k, v := range set.Placeholders {
				if g, ok := got[k]; !ok || g != v {
					t.Errorf("%s: %s = %v (set %v), PoB %v", set.Inputs, k, g, ok, v)
				}
			}
			for k, v := range got {
				if _, ok := set.Placeholders[k]; !ok {
					t.Errorf("%s: %s = %v, PoB has none", set.Inputs, k, v)
				}
			}
		}
	}
}

func oracleConfigValue(v any) calculator2.ConfigValue {
	switch v := v.(type) {
	case bool:
		return calculator2.ConfigValue{Boolean: v, IsBoolean: true}
	case float64:
		return calculator2.ConfigValue{Number: v, IsNumber: true}
	case string:
		return calculator2.ConfigValue{String: v, IsString: true}
	}
	return calculator2.ConfigValue{}
}

// The differences behind three recorded tooltips: the custom group's "-32 Total Life", Onslaught's "+0.29 Cast Rate"
// and Uber's "-70.0%" Hit DPS.
func TestConfigCompareMatchesPoB(t *testing.T) {
	calc := calculatedFixture(t, "sorceress_spark.xml")
	o := readConfigWhatIfOracle(t)
	if len(o.Custom) != 1 || !strings.Contains(strings.Join(o.Custom[0].Lines, "\n"), "-32 Total Life") {
		t.Fatalf("custom oracle %+v", o.Custom)
	}
	v, err := calc.CustomModGroupCompare(0)
	mustOK(t, err)
	if len(v.Blocks) != 1 {
		t.Fatalf("%d blocks, want 1", len(v.Blocks))
	}
	if d := v.Blocks[0].Compare.Output["Life"] - v.Base.Output["Life"]; d != -32 {
		t.Errorf("custom group off: Life %+v, PoB -32", d)
	}
	if v, err := calc.CustomModGroupCompare(1); err != nil || len(v.Blocks) != 0 {
		t.Errorf("no group 1: %d blocks, %v", len(v.Blocks), err)
	}

	onslaught, err := calc.ConfigCompare("buffOnslaught", oracleConfigValue(true))
	mustOK(t, err)
	if d := math.Round((onslaught.Blocks[0].Compare.Output["Speed"]-onslaught.Base.Output["Speed"])*100) / 100; d != 0.29 {
		t.Errorf("Onslaught: Cast Rate %+v, PoB +0.29", d)
	}
	uber, err := calc.ConfigCompare("enemyIsBoss", oracleConfigValue("Uber"))
	mustOK(t, err)
	if pc := math.Round((uber.Blocks[0].Compare.Output["TotalDPS"]/uber.Base.Output["TotalDPS"]*100-100)*10) / 10; pc != -70 {
		t.Errorf("Uber: Hit DPS %+v%%, PoB -70.0%%", pc)
	}
	// the build itself is left as it was
	after := recalc(t, calc)
	if after.Output["Life"] != v.Base.Output["Life"] {
		t.Errorf("Life %v after the comparisons, %v before", after.Output["Life"], v.Base.Output["Life"])
	}
}
