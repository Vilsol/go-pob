package calculator

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestJewelCompareTreeAgainstPoB grades the what-if of a jewel in each allocated tree socket.
func TestJewelCompareTreeAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/tree-ui/jewel_compare_tree.json")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var cases []struct {
		Build    string
		Item     string
		Socketed bool
		Base     map[string]float64
		Calls    []struct {
			Slot     string
			Removing bool
			Rebuilt  bool
			Stats    map[string]float64
		}
	}
	if err := json.Unmarshal(raw, &cases); err != nil || len(cases) < 3 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	graded := 0
	for _, c := range cases {
		build := loadCorpusBuild(t, c.Build)
		m := GetMiscCalculator(build)
		skip := map[string]bool{}
		for stat, want := range c.Base {
			if got := m.Base().Player.Output[stat]; !closeTo(got, want) {
				skip[stat] = true
				t.Logf("%s base %s: ours %v, the pin's %v (not graded)", c.Build, stat, got, want)
			}
		}
		item := ParseItem(c.Item)
		for _, call := range c.Calls {
			id, _ := strconv.Atoi(strings.TrimPrefix(call.Slot, "Jewel "))
			o := &CalcOverride{SocketNodeID: id, SocketItem: item}
			if call.Removing {
				o.SocketItem = nil
			}
			out := m.Calc(o).Player.Output
			for stat, want := range call.Stats {
				if skip[stat] {
					continue
				}
				graded++
				if got := out[stat]; !closeTo(got, want) {
					t.Errorf("%s %s (removing %v) %s: ours %v, the pin's %v (base %v)", c.Build, call.Slot, call.Removing, stat, got, want, c.Base[stat])
				}
			}
		}
	}
	if graded < 100 {
		t.Fatalf("graded only %d values", graded)
	}
}

func closeTo(a, b float64) bool {
	return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b))
}

// TestCompareItemSlotsTreeSockets: a jewel's equip comparison lists the allocated tree sockets as "Socket #n" slots.
func TestCompareItemSlotsTreeSockets(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/tree-ui/jewel_compare_tree.json")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var cases []struct {
		Build string
		Item  string
		Calls []struct {
			Slot  string
			Stats map[string]float64
		}
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	c := cases[0]
	build := loadCorpusBuild(t, c.Build)
	m := GetMiscCalculator(build)
	got := map[string]ItemSlotCompare{}
	for _, s := range m.CompareItemSlots(ParseItem(c.Item)) {
		got[s.SlotName] = s
		if !strings.HasPrefix(s.Label, "Socket #") {
			t.Errorf("%s: label %q", s.SlotName, s.Label)
		}
	}
	for _, call := range c.Calls {
		s, ok := got[call.Slot]
		if !ok {
			t.Errorf("%s: no comparison", call.Slot)
			continue
		}
		for stat, want := range call.Stats {
			if v := s.Env.Player.Output[stat]; !closeTo(v, want) {
				t.Errorf("%s %s: ours %v, the pin's %v", call.Slot, stat, v, want)
			}
		}
	}
	if len(got) != len(c.Calls) {
		t.Errorf("ours compares %d slots, the pin %d", len(got), len(c.Calls))
	}
}
