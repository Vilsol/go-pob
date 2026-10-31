package exposition

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
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
