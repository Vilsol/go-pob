//go:build poe2gamedata

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/loader"
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

// TestPoE2ItemBasesAreThePins: bases.lua over go-pob-data's 0.5 tables, .it files and stat descriptions, with
// the pin's Export/Bases templates, writes the pin's src/Data/Bases/*.lua.
func TestPoE2ItemBasesAreThePins(t *testing.T) {
	poe2PinFile(t, "Data/Misc.lua")
	files, x, err := poe2ItemBaseFiles(os.Getenv("POB2_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range poe2BaseItemTypeNames {
		rel := "Data/Bases/" + name + ".lua"
		t.Run(name, func(t *testing.T) {
			want := poe2PinFile(t, rel)
			if name == "flask" {
				// the charms' buff lines need UtilityFlaskBuffs and BuffDefinitions, which go-pob-data does not publish
				if n := len(poe2CharmBuffRe.FindAllString(want, -1)); n != x.charmBuffs || n == 0 {
					t.Fatalf("the pin's flask.lua has %d charm buff lines, the dumper skipped %d", n, x.charmBuffs)
				}
				want = poe2CharmBuffRe.ReplaceAllString(want, "")
			}
			firstLineDiff(t, rel, files[rel], want)
		})
	}
}

var poe2CharmBuffRe = regexp.MustCompile(`buff = \{ "[^\n]*" \}, `)

var poe2TradeHashesRe = regexp.MustCompile(`tradeHashes = \{ ((?:\[\d+\] = \{ [^\n]*? \}, )*) \},`)
var poe2TradeHashRe = regexp.MustCompile(`\[(\d+)\] = \{ [^\n]*? \}, `)

// poe2SortTradeHashes puts each tradeHashes table's entries in key order: soulcores.lua writes them in
// pairs() order.
func poe2SortTradeHashes(text string) string {
	return poe2TradeHashesRe.ReplaceAllStringFunc(text, func(m string) string {
		inner := poe2TradeHashesRe.FindStringSubmatch(m)[1]
		entries := poe2TradeHashRe.FindAllString(inner, -1)
		sort.Slice(entries, func(i, j int) bool {
			a, _ := strconv.ParseUint(poe2TradeHashRe.FindStringSubmatch(entries[i])[1], 10, 64)
			b, _ := strconv.ParseUint(poe2TradeHashRe.FindStringSubmatch(entries[j])[1], 10, 64)
			return a < b
		})
		return "tradeHashes = { " + strings.Join(entries, "") + " },"
	})
}

// TestPoE2SoulCoreTradeHashes: murmurHash2 over a soul core stat's go-pob-data hash gives a trade hash
// the pin's src/Data/ModRunes.lua carries.
func TestPoE2SoulCoreTradeHashes(t *testing.T) {
	src := poe2PinFile(t, "Data/ModRunes.lua")
	pinHashes := map[string]bool{}
	for _, m := range poe2TradeHashRe.FindAllStringSubmatch(src, -1) {
		pinHashes[m[1]] = true
	}
	stats, err := poe2Table[poe2HashedStat]("Stats")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := poe2Table[poe2SoulCoreStat]("SoulCoreStats")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, r := range rows {
		if len(r.Stats) != 1 || strings.Contains(stats[r.Stats[0]].ID, "minimum") || r.StatValue[0] == 0 {
			continue
		}
		h := strconv.FormatUint(uint64(murmurHash2(intToBytes(stats[r.Stats[0]].Hash), 0x02312233)), 10)
		if pinHashes[h] {
			checked++
		}
	}
	t.Logf("%d single-stat soul core rows hash to one of the pin's %d trade hashes", checked, len(pinHashes))
	if checked < len(pinHashes)/2 {
		t.Fatalf("%d single-stat soul core rows hash to one of the pin's %d trade hashes", checked, len(pinHashes))
	}
}

// TestPoE2ModRunesIsThePins: soulcores.lua over go-pob-data's 0.5 tables writes the pin's src/Data/ModRunes.lua,
// up to the order of each tradeHashes table.
func TestPoE2ModRunesIsThePins(t *testing.T) {
	want := poe2PinFile(t, "Data/ModRunes.lua")
	for _, table := range []string{"SoulCoreTypes", "SoulCoreLimits", "SoulCoreStatCategories"} {
		if ok, err := loader.HasTable(gameDataContext(), poe2Version, table); err != nil {
			t.Fatal(err)
		} else if !ok {
			t.Skipf("go-pob-data %s does not publish %s", poe2Version, table)
		}
	}
	got, err := poe2ModRunesLua(os.Getenv("POB2_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	firstLineDiff(t, "Data/ModRunes.lua", got, poe2SortTradeHashes(want))
}
