package calculator

import (
	"fmt"
	"strings"
	"testing"
)

// TestSecondaryAscendancyList: Build.lua:1093-1138 over the pin's 3_29 tree.
func TestSecondaryAscendancyList(t *testing.T) {
	want := "None:0|Abyssal Bloodline:16|Aul Bloodline:8|Breachlord Bloodline:13|Catarina Bloodline:7|Chaos Bloodline:4|" +
		"Delirious Bloodline:12|Farrul Bloodline:11|Lycia Bloodline:9|Nameless Bloodline:6|Olroth Bloodline:10|" +
		"Oshabi Bloodline:5|Saresh Bloodline:14|Velka Bloodline:15"
	build := loadCorpusBuild(t, "0030-saboteur-cyclone-2.xml")
	spec := &build.Tree.Specs[build.ActiveSpecIndex()]
	for _, sel := range []int{0, 1, 9} {
		spec.SecondaryAscendClassID = &sel
		got := SecondaryAscendancyList(build)
		parts := make([]string, 0, len(got.Entries))
		for _, e := range got.Entries {
			parts = append(parts, fmt.Sprintf("%s:%d", e.Label, e.AscendClassID))
		}
		if s := strings.Join(parts, "|"); s != want {
			t.Errorf("selection %d:\n got %s\nwant %s", sel, s, want)
		}
		if got.Selected != sel || !got.Enabled {
			t.Errorf("selection %d: selected %d enabled %v", sel, got.Selected, got.Enabled)
		}
	}
}
