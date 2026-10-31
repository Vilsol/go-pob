package exposition

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

func calculateFile(t *testing.T, path string) *Outputs {
	t.Helper()
	xml, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return out
}

func assertClose(t *testing.T, build string, stat string, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
		t.Errorf("%s %s: PoB-PoE2 %v, ours %v", build, stat, want, got)
	}
}

// The values are PoB-PoE2's for the fixture, from oracle/dump_build.lua run on the pinned checkout.
func TestFixtureBuildMatchesPoB(t *testing.T) {
	out := calculateFile(t, filepath.Join("testdata", "sorceress_spark.xml"))
	for stat, want := range map[string]float64{
		"Life":       683,
		"Mana":       231,
		"Spirit":     100,
		"Armour":     928,
		"FireResist": -15,
		"AverageHit": 17.44,
		"TotalDPS":   24.914285714286,
	} {
		assertClose(t, "sorceress_spark.xml", stat, out.Output[stat], want)
	}
	if out.ClassName != "Sorceress" || out.AscendClassName != "None" || out.Level != 40 {
		t.Errorf("class %q ascendancy %q level %d, want Sorceress None 40", out.ClassName, out.AscendClassName, out.Level)
	}
}

func TestCorpusBuildsMatchOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	corpusDir := filepath.Join("..", "..", "testdata", "corpus-poe2-0.5")
	oraclePath := filepath.Join("..", "..", "testdata", "oracle-poe2-0.5", "output-poe2-0.5.jsonl.gz")
	f, err := os.Open(oraclePath)
	if err != nil {
		t.Skipf("no PoE2 oracle: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	for n := 0; n < 3; n++ {
		if !sc.Scan() {
			t.Fatalf("oracle ended after %d builds: %v", n, sc.Err())
		}
		var rec struct {
			Build  string             `json:"build"`
			Output map[string]float64 `json:"output"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatal(err)
		}
		out := calculateFile(t, filepath.Join(corpusDir, rec.Build))
		for _, stat := range []string{"Life", "TotalDPS"} {
			want, ok := rec.Output[stat]
			if !ok {
				t.Fatalf("%s: oracle has no %s", rec.Build, stat)
			}
			assertClose(t, rec.Build, stat, out.Output[stat], want)
		}
	}
}

func parseFile(t *testing.T, path string) *Calculator {
	t.Helper()
	xml, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return calc
}

func recalc(t *testing.T, calc *Calculator) *Outputs {
	t.Helper()
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// The values are PoB-PoE2's for the fixture saved with resistancePenalty 0 (Act 1), from
// oracle/dump_build.lua run on the pinned checkout.
func TestSetConfigInputRecalculates(t *testing.T) {
	calc := parseFile(t, filepath.Join("testdata", "sorceress_spark.xml"))
	recalc(t, calc)
	calc.SetConfigInput("resistancePenalty", calculator2.ConfigValue{Number: 0, IsNumber: true}, false)
	out := recalc(t, calc)
	assertClose(t, "resistancePenalty 0", "FireResist", out.Output["FireResist"], 45)
	assertClose(t, "resistancePenalty 0", "LightningResist", out.Output["LightningResist"], 10)
	if v, ok := out.ConfigInputs["resistancePenalty"]; !ok || !v.IsNumber || v.Number != 0 {
		t.Errorf("config inputs %v, want resistancePenalty = 0", out.ConfigInputs)
	}

	calc.SetConfigInput("resistancePenalty", calculator2.ConfigValue{Number: -60, IsNumber: true}, true)
	out = recalc(t, calc)
	assertClose(t, "resistancePenalty default", "FireResist", out.Output["FireResist"], -15)
	if _, ok := out.ConfigInputs["resistancePenalty"]; ok {
		t.Errorf("a default-valued input stays saved: %v", out.ConfigInputs)
	}
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(saved, "resistancePenalty") {
		t.Errorf("the saved build still carries the default-valued input")
	}
}

// The values are PoB-PoE2's for the fixture with each main skill selected (oracle/dump_build.lua),
// and the labels are its socketGroupList displayLabel and main skill dropdown.
func TestMainSkillSelectionRecalculates(t *testing.T) {
	calc := parseFile(t, filepath.Join("testdata", "sorceress_three_spells.xml"))
	out := recalc(t, calc)
	assertClose(t, "Spark", "TotalDPS", out.Output["TotalDPS"], 24.914285714286)
	want := []calculator2.SocketGroupView{{Label: "Spark, Fireball", Skills: []string{"Spark", "Fireball"}}, {Label: "Arc", Skills: []string{"Arc"}}}
	if !reflect.DeepEqual(out.SocketGroups, want) || out.MainSocketGroup != 1 || out.MainActiveSkill != 1 {
		t.Errorf("socket groups %+v main %d/%d, want %+v main 1/1", out.SocketGroups, out.MainSocketGroup, out.MainActiveSkill, want)
	}

	calc.SetMainActiveSkill(2)
	out = recalc(t, calc)
	assertClose(t, "Fireball", "TotalDPS", out.Output["TotalDPS"], 32.76875)
	assertClose(t, "Fireball", "AverageHit", out.Output["AverageHit"], 39.3225)
	assertClose(t, "Fireball", "ManaCost", out.Output["ManaCost"], 35)
	if out.MainActiveSkill != 2 {
		t.Errorf("main active skill %d, want 2", out.MainActiveSkill)
	}

	calc.SetMainActiveSkill(1)
	calc.SetMainSocketGroup(2)
	out = recalc(t, calc)
	assertClose(t, "Arc", "TotalDPS", out.Output["TotalDPS"], 25.020454545455)
	assertClose(t, "Arc", "AverageHit", out.Output["AverageHit"], 27.5225)
	if out.MainSocketGroup != 2 || !out.SkillFlags["spell"] {
		t.Errorf("main socket group %d, skill flags %v; want 2 and a spell", out.MainSocketGroup, out.SkillFlags)
	}
}
