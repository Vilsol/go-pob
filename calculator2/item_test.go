package calculator2

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func readOracleRows(t *testing.T, path string) []*oracleItem {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // test data
	if err != nil {
		t.Fatal(err)
	}
	var rows []*oracleItem
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64<<20), 64<<20)
	for sc.Scan() {
		row := &oracleItem{}
		if err := json.Unmarshal(sc.Bytes(), row); err != nil {
			t.Fatalf("bad oracle row: %v", err)
		}
		rows = append(rows, row)
	}
	return rows
}

// TestItemFixtureVsPoBPoE2 loads the hand-made items in testdata/item_fixture.xml against the rows
// PoB-PoE2 printed for them (testdata/item_fixture_pob.jsonl, regenerated and checked by
// TestItemFixtureOracleIsCurrent), with PoB's own parse of every line as the mod source.
func TestItemFixtureVsPoBPoE2(t *testing.T) {
	rows := readOracleRows(t, filepath.Join("testdata", "item_fixture_pob.jsonl"))
	if len(rows) != 10 {
		t.Fatalf("the fixture holds %d items, want 10", len(rows))
	}
	g := gradeItems(rows, nil)
	g.report(t)
	if g.agree != g.items {
		t.Errorf("%d of %d fixture items disagree with PoB-PoE2", g.items-g.agree, g.items)
	}
}

// TestItemFixtureOracleIsCurrent re-runs the item oracle on the fixture under the PoB-PoE2 pin and
// requires the committed rows to match.
func TestItemFixtureOracleIsCurrent(t *testing.T) {
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit; set OPERATOR=true")
	}
	xml, err := filepath.Abs(filepath.Join("testdata", "item_fixture.xml"))
	if err != nil {
		t.Fatal(err)
	}
	fresh := runItemOracle(t, []string{xml})
	committed := readOracleRows(t, filepath.Join("testdata", "item_fixture_pob.jsonl"))
	if len(fresh) != len(committed) {
		t.Fatalf("the oracle printed %d rows, the fixture holds %d", len(fresh), len(committed))
	}
	for n := range fresh {
		a, _ := json.Marshal(fresh[n])
		b, _ := json.Marshal(committed[n])
		if !bytes.Equal(a, b) {
			t.Errorf("row %d (item %d) differs from PoB-PoE2's current output; regenerate testdata/item_fixture_pob.jsonl", n, fresh[n].ID)
		}
	}
}

// TestApplyRangeScalabilityVsPoB runs applyRange's data.modScalability branch on entries copied from
// the PoB-PoE2 pin; the expected lines are PoB's own itemLib.applyRange results.
func TestApplyRangeScalabilityVsPoB(t *testing.T) {
	entries := map[string][]ModScalability{
		"Skills have # seconds to Cooldown": {{IsScalable: true, Formats: []string{"milliseconds_to_seconds_2dp_if_required"}}},
		"#% increased Armour":               {{IsScalable: true}},
		"Adds # to # Physical Damage":       {{IsScalable: true}, {IsScalable: true}},
	}
	saved := modScalabilityFor
	t.Cleanup(func() { modScalabilityFor = saved })
	modScalabilityFor = func(key string) ([]ModScalability, bool) {
		e, ok := entries[key]
		return e, ok
	}
	noParse := func(string) ([]mod.Mod, string) { return nil, "" }
	for _, c := range []struct {
		line          string
		rangeVal, vsc float64
		want          string
	}{
		{"Skills have -2.00 seconds to Cooldown", 1, 1, "Skills have -2 seconds to Cooldown"},
		{"(10-20)% increased Armour", 0.5, 1.2, "18% increased Armour"},
		{"Adds (5-10) to (20-30) Physical Damage", 0.3, 1, "Adds 7 to 23 Physical Damage"},
	} {
		vsc := c.vsc
		if got := applyRange(c.line, c.rangeVal, nil, &vsc, nil, noParse); got != c.want {
			t.Errorf("applyRange(%q, %v, %v) = %q, PoB %q", c.line, c.rangeVal, c.vsc, got, c.want)
		}
	}
}
