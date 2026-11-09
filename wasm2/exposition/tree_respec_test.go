package exposition

import (
	"os"
	"slices"
	"testing"
)

// The tree list tooltip's respec cost and a node tooltip's gold line, as PoB-PoE2 shows them for warrior_ascendancy
// (level 80): switching to "Respec" costs "37,752 Gold" for 3 passive and 2 ascendancy nodes, and unallocating the
// ascendancy node 27418 "14,520 Gold".
func TestTreeRespecGold(t *testing.T) {
	xml, err := os.ReadFile("../../calculator2/testdata/warrior_ascendancy.xml")
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	rows, err := calc.TreeSpecRows()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("%d specs, want 2", len(rows))
	}
	if r := rows[0]; r.RespecGold != 0 || r.RespecPassives != 0 || r.RespecAscendancy != 0 {
		t.Errorf("the active spec has a respec cost: %+v", r)
	}
	if r := rows[1]; r.RespecGold != 37752 || r.RespecPassives != 3 || r.RespecAscendancy != 2 {
		t.Errorf("Respec: %v gold, %d passives, %d ascendancy; PoB-PoE2 37752, 3, 2", r.RespecGold, r.RespecPassives, r.RespecAscendancy)
	}
	tip, err := calc.NodeTip(27418, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tip.PathLines, "14520 Gold required to unallocate this node") {
		t.Errorf("node 27418 lines %q; PoB-PoE2 shows 14,520 Gold required to unallocate this node", tip.PathLines)
	}
}
