package calculator2

import (
	"reflect"
	"testing"
)

// The expectation is LuaJIT's table.sort of the same keys by key descending, which ties in its own order.
func TestLuaSortMatchesLuaJIT(t *testing.T) {
	keys := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5, 8, 9, 7, 9, 3, 2, 3, 8, 4, 6, 2, 6, 4, 3, 3, 8, 3, 2, 7, 9, 5}
	type row struct{ k, id int }
	rows := make([]row, len(keys))
	for i, k := range keys {
		rows[i] = row{k, i + 1}
	}
	luaSort(rows, func(a, b row) bool { return a.k > b.k })
	got := make([]int, len(rows))
	for i, r := range rows {
		got[i] = r.id
	}
	want := []int{6, 31, 15, 13, 27, 12, 19, 30, 14, 8, 21, 23, 11, 9, 5, 32, 24, 20, 3, 26, 28, 25, 16, 18, 10, 1, 7, 17, 29, 22, 2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("luaSort order %v, LuaJIT %v", got, want)
	}
}

// CalcBreakdownControl.lua:181-191 (PoB-PoE2): a breakdown with a source sorts its rows by reqNum, highest first.
func TestBreakdownViewSortsRequirementRows(t *testing.T) {
	b := &Breakdown{Entries: map[string]*BreakdownEntry{"ReqStr": {
		Source:  true,
		ColList: []BreakdownCol{{"Str", "req"}, {"Source", "source"}},
		RowList: []map[string]string{{"req": "11", "reqNum": "11", "source": "Item"}, {"req": "^xDD002265", "reqNum": "65", "source": "Gem"}},
	}}}
	v, ok := b.View("ReqStr")
	if !ok {
		t.Fatal("no view")
	}
	want := []map[string]string{{"req": "^xDD002265", "source": "Gem"}, {"req": "11", "source": "Item"}}
	if !reflect.DeepEqual(v.Tables[0].Rows, want) {
		t.Errorf("rows %v, want %v", v.Tables[0].Rows, want)
	}
}
