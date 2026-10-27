//go:build poe2gamedata

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func poe2PinFile(t *testing.T, rel string) string {
	t.Helper()
	dir := os.Getenv("POB2_DIR")
	if dir == "" {
		t.Skip("POB2_DIR is not set; run `task poe2-gamedata-check`, which points it at the pinned PoB-PoE2")
	}
	b, err := os.ReadFile(filepath.Join(dir, "src", rel)) //nolint:gosec // the pinned checkout's own file
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func firstLineDiff(t *testing.T, what, got, want string) {
	t.Helper()
	if got == want {
		return
	}
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			t.Fatalf("%s differs from the pin's at line %d:\n got  %.300q\n want %.300q", what, i+1, gl, wl)
		}
	}
}

// TestPoE2MiscLuaIsThePinsMisc: miscdata.lua over go-pob-data's 0.5 tables writes the pin's src/Data/Misc.lua.
func TestPoE2MiscLuaIsThePinsMisc(t *testing.T) {
	want := poe2PinFile(t, "Data/Misc.lua")
	got, err := poe2MiscLua()
	if err != nil {
		t.Fatal(err)
	}
	firstLineDiff(t, "Data/Misc.lua", got, want)
}

func luaStrings(t *testing.T, v any) []string {
	t.Helper()
	tbl, ok := v.(*luaTable)
	if !ok {
		if v != nil {
			t.Fatalf("%v is not a table", v)
		}
		return nil
	}
	out := make([]string, 0, len(tbl.arr))
	for _, s := range tbl.arr {
		out = append(out, s.(string))
	}
	return out
}

type poe2AlternatePassiveAddition struct {
	ID       string `json:"Id"`
	Stats    []int  `json:"Stats"`
	Stat1Min int    `json:"Stat1Min"`
	Stat1Max int    `json:"Stat1Max"`
	Stat2Min int    `json:"Stat2Min"`
	Stat2Max int    `json:"Stat2Max"`
	Stat3Min int    `json:"Stat3Min"`
	Stat3Max int    `json:"Stat3Max"`
}

// TestPoE2LegionPassivesAreThePins: legionPassives.lua over go-pob-data's 0.5 tables and stat descriptions
// gives the pin's src/Data/TimelessJewelData/LegionPassives.lua nodes, and its additions' stat lines.
func TestPoE2LegionPassivesAreThePins(t *testing.T) {
	src := poe2PinFile(t, "Data/TimelessJewelData/LegionPassives.lua")
	pinNodes, err := luaTableLiteral(src, "\n\t[\"nodes\"] = ")
	if err != nil {
		t.Fatal(err)
	}
	pinAdditions, err := luaTableLiteral(src, "\n\t[\"additions\"] = ")
	if err != nil {
		t.Fatal(err)
	}
	dump, err := dumpPoE2LegionPassives()
	if err != nil {
		t.Fatal(err)
	}
	nodes := dump.([]poe2LegionNode)
	want := pinNodes.arr
	if len(nodes) != len(want) {
		t.Fatalf("%d nodes, the pin has %d", len(nodes), len(want))
	}
	for i, n := range nodes {
		w := want[i].(*luaTable).hash
		got := fmt.Sprintf("%s|%s|%q|%v|%v|%v", n.ID, n.Dn, n.Sd, n.Ks, n.Not, n.M)
		exp := fmt.Sprintf("%s|%s|%q|%v|%v|%v", w["id"], w["dn"], luaStrings(t, w["sd"]), w["ks"], w["not"], w["m"])
		if got != exp {
			t.Errorf("node %d:\n got  %s\n want %s", i+1, got, exp)
		}
	}

	stats, err := poe2Table[poe2Stat]("Stats")
	if err != nil {
		t.Fatal(err)
	}
	additions, err := poe2Table[poe2AlternatePassiveAddition]("AlternatePassiveAdditions")
	if err != nil {
		t.Fatal(err)
	}
	describer, err := poe2LoadStatFile("passive_skill_stat_descriptions")
	if err != nil {
		t.Fatal(err)
	}
	wantAdds := pinAdditions.arr
	if len(additions) != len(wantAdds) {
		t.Fatalf("%d additions, the pin has %d", len(additions), len(wantAdds))
	}
	for i, a := range additions {
		sd, err := poe2LegionStats(describer, stats, a.ID, a.Stats, [][2]int{{a.Stat1Min, a.Stat1Max}, {a.Stat2Min, a.Stat2Max}, {a.Stat3Min, a.Stat3Max}})
		if err != nil {
			t.Fatal(err)
		}
		w := wantAdds[i].(*luaTable).hash
		if got, exp := fmt.Sprintf("%s|%q", fixLegionDatErrors(a.ID, ""), sd), fmt.Sprintf("%s|%q", w["id"], luaStrings(t, w["sd"])); got != exp {
			t.Errorf("addition %d:\n got  %s\n want %s", i+1, got, exp)
		}
	}
}
