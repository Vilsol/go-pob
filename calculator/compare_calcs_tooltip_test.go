package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestCompareModRows: The tooltip's rows are the breakdown's modifier rows, keyed without the item's id (CompareCalcsHelpers.lua:111-119).
func TestCompareModRows(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0001-necromancer-unknown.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
	sd := ModSectionData{ModName: []string{"Life", "Mana", "EnergyShield", "Damage"}, Multi: true}
	rows := env.CompareModRows(sd, false)
	section := env.CalcsModSection(sd, false)
	if len(rows) == 0 || len(rows) != len(section) {
		t.Fatalf("%d rows, the breakdown has %d", len(rows), len(section))
	}
	items := 0
	for _, r := range rows {
		if r.SourceType == "Item" {
			items++
			if !strings.HasPrefix(r.Key, "Item:") || strings.HasPrefix(strings.TrimPrefix(r.Key, "Item:"), "-1") {
				t.Errorf("item key %q keeps its id", r.Key)
			}
			if r.SourceName == "" {
				t.Errorf("item row %q names no item", r.Key)
			}
		}
		if !strings.HasPrefix(r.ModName, "  ") || r.Display == "" {
			t.Errorf("row %+v", r)
		}
	}
	if items < 3 {
		t.Fatalf("only %d item rows: no control", items)
	}
}
