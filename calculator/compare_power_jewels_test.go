package calculator

import (
	"strings"
	"testing"
)

// TestCompareJewelPowerBuilder: CompareTab.lua:2785-2868: each jewel B holds in a tree socket where A has another.
func TestCompareJewelPowerBuilder(t *testing.T) {
	initItemsTabData(t)
	a := loadCorpusBuild(t, "0011-saboteur-cyclone.xml")
	b := loadCorpusBuild(t, "0009-trickster-cobra-lash.xml")
	slots := CompareJewelSlots(a, b)
	var want []CompareJewelSlot
	for _, j := range slots {
		if j.CText != "" && (j.PText == "" || PeekItem(j.PText).Name != PeekItem(j.CText).Name) {
			want = append(want, j)
		}
	}
	if len(want) == 0 {
		t.Fatalf("REFUSING: B holds no jewel A lacks (%d slots)", len(slots))
	}
	builder := NewCompareJewelPowerBuilder(a, b, []string{"Life", "FullDPS"})
	for builder.Step(1<<62) < 1 {
	}
	report := builder.Report()
	if len(report) != len(want) {
		t.Fatalf("%d rows for %d jewels: %+v", len(report), len(want), report)
	}
	misc := GetMiscCalculator(a)
	life, _ := powerStatByName("Life")
	base := powerStatValue(misc.Calc(nil), life)
	allocatedSeen := 0
	for k, e := range report {
		cut := strings.LastIndex(e.Name, ", ")
		name, label := e.Name[:max(cut, 0)], e.Name[cut+2:]
		j := want[k]
		if name != PeekItem(j.CText).Name || e.Category != "Item" || len(e.Impact) != 2 {
			t.Fatalf("unexpected row %+v", e)
		}
		if j.PAllocated {
			allocatedSeen++
			if label != j.Label {
				t.Errorf("%s: label %q, want %q", name, label, j.Label)
			}
			item := ParseItem(j.CText)
			item.NormaliseQuality()
			if got := powerStatValue(misc.Calc(&CalcOverride{SocketNodeID: j.NodeID, SocketItem: item}), life) - base; got != e.Impact[0] {
				t.Errorf("%s in its socket: Life %v, the what-if %v", name, e.Impact[0], got)
			}
		} else if label != j.Label+" (best socket)" {
			t.Errorf("%s: label %q, want %q", name, label, j.Label+" (best socket)")
		}
	}
	t.Logf("%d jewel rows, %d measured in a socket A allocates", len(report), allocatedSeen)
	if allocatedSeen == 0 || allocatedSeen == len(report) {
		t.Fatalf("REFUSING: %d of %d rows in a socket A allocates: both kinds are needed", allocatedSeen, len(report))
	}
}
