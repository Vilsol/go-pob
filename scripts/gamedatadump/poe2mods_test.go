//go:build poe2gamedata

package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	poe2ModRowRe         = regexp.MustCompile(`^\t\["([^"]+)"\] = \{ `)
	poe2ModTradeHashesRe = regexp.MustCompile(`tradeHashes = \{ (.*)\} \},$`)
	poe2ModTradeHashRe   = regexp.MustCompile(`\[(\d+)\] = \{ .*? \}, `)
)

// poe2SortModTradeHashes puts a mods.lua row's tradeHashes entries in key order: mods.lua writes them in pairs() order.
func poe2SortModTradeHashes(line string) string {
	m := poe2ModTradeHashesRe.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	entries := poe2ModTradeHashRe.FindAllString(line[m[2]:m[3]], -1)
	sort.Slice(entries, func(i, j int) bool {
		a, _ := strconv.ParseUint(poe2ModTradeHashRe.FindStringSubmatch(entries[i])[1], 10, 64)
		b, _ := strconv.ParseUint(poe2ModTradeHashRe.FindStringSubmatch(entries[j])[1], 10, 64)
		return a < b
	})
	return line[:m[2]] + strings.Join(entries, "") + line[m[3]:]
}

func poe2ModRows(text string) ([]string, map[string]string) {
	var ids []string
	rows := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if m := poe2ModRowRe.FindStringSubmatch(line); m != nil {
			ids = append(ids, m[1])
			rows[m[1]] = poe2SortModTradeHashes(line)
		}
	}
	return ids, rows
}

// TestPoE2ModsAreThePins: mods.lua over go-pob-data's 0.5 tables and stat descriptions writes the pin's
// src/Data/Mod*.lua, row for row, up to the order of each tradeHashes table.
func TestPoE2ModsAreThePins(t *testing.T) {
	poe2PinFile(t, "Data/ModItem.lua")
	files, err := poe2ModsLua()
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range files {
		t.Run(name, func(t *testing.T) {
			want := poe2PinFile(t, name)
			wantIDs, wantRows := poe2ModRows(want)
			gotIDs, gotRows := poe2ModRows(got)
			same, shown := 0, 0
			for _, id := range wantIDs {
				if gotRows[id] == wantRows[id] {
					same++
				} else if shown < 5 {
					shown++
					t.Errorf("%s:\n got  %.400q\n want %.400q", id, gotRows[id], wantRows[id])
				}
			}
			extra := 0
			for _, id := range gotIDs {
				if _, ok := wantRows[id]; !ok {
					extra++
				}
			}
			t.Logf("%s: %d / %d rows identical, %d rows the pin does not carry", name, same, len(wantIDs), extra)
			if same != len(wantIDs) || extra != 0 || len(gotIDs) != len(wantIDs) {
				t.Fail()
				return
			}
			if strings.Join(gotIDs, ",") != strings.Join(wantIDs, ",") {
				t.Fatal("the rows are in another order")
			}
		})
	}
}

// poe2ScalabilityTextDrift maps a pin row to ours where go-pob-data's 0.5 stat descriptions carry a later
// wording than the pin's export; neither has a placeholder, so no item line can reach them.
var poe2ScalabilityTextDrift = map[string]string{
	"\t[\"Defeating the Arbiter grants completion of a section of the Fortress\"] = {  },": "\t[\"Defeating the Arbiter of Divinity as the Map Owner twice will complete all uncompleted Map Nodes in the Fortress\"] = {  },",
}

// TestPoE2ModScalabilityIsThePins: modScalability.lua over go-pob-data's 0.5 Stats table and stat descriptions
// writes the pin's src/Data/ModScalability.lua, row for row, up to poe2ScalabilityTextDrift.
func TestPoE2ModScalabilityIsThePins(t *testing.T) {
	want := poe2PinFile(t, "Data/ModScalability.lua")
	got, err := poe2ModScalabilityLua()
	if err != nil {
		t.Fatal(err)
	}
	wantRows, gotRows := strings.Split(want, "\n"), strings.Split(got, "\n")
	gotSet := map[string]bool{}
	for _, row := range gotRows {
		gotSet[row] = true
	}
	same, drift := 0, 0
	for _, row := range wantRows {
		ours, drifted := poe2ScalabilityTextDrift[row]
		switch {
		case gotSet[row]:
			same++
		case drifted && gotSet[ours]:
			drift++
		default:
			t.Errorf("the pin's row is missing: %.400q", row)
		}
	}
	t.Logf("Data/ModScalability.lua: %d / %d rows identical, %d in poe2ScalabilityTextDrift, %d rows written", same, len(wantRows), drift, len(gotRows))
	if len(gotRows) != len(wantRows) || drift != len(poe2ScalabilityTextDrift) {
		t.Errorf("%d rows written and %d drifted rows; the pin has %d rows and %d are listed", len(gotRows), drift, len(wantRows), len(poe2ScalabilityTextDrift))
	}
}
