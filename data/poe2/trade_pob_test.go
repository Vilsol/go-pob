package poe2

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type pobTradeData struct {
	TradeStats       []TradeStatCategory `json:"tradeStats"`
	StatDescriptions []struct {
		Stats []string `json:"stats"`
		Forms []struct {
			Text  string   `json:"text"`
			Limit [][2]any `json:"limit"`
			Flags []struct {
				V any    `json:"v"`
				K string `json:"k"`
			} `json:"flags"`
		} `json:"forms"`
	} `json:"statDescriptions"`
}

// pobTradeDataRun is testdata/trade_data.lua's dump of the pin's Data/TradeSiteStats.lua and
// Data/StatDescriptions/stat_descriptions.lua.
func pobTradeDataRun(t *testing.T) pobTradeData {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit; set OPERATOR=true")
	}
	common, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Skipf("SKIP: not in a git checkout: %v", err)
	}
	main := filepath.Dir(strings.TrimSpace(string(common)))
	pob := os.Getenv("POB2_DIR")
	if pob == "" {
		pob = filepath.Join(main, "..", "PathOfBuilding-PoE2")
	}
	luajit := filepath.Join(main, ".luajit-det", "bin", "luajit")
	for _, p := range []string{filepath.Join(pob, "src", "Data", "TradeSiteStats.lua"), luajit} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("SKIP: %s is absent", p)
		}
	}
	probe, err := filepath.Abs(filepath.Join("testdata", "trade_data.lua"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), luajit, probe)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(), "LUA_PATH="+pob+"/runtime/lua/?.lua;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the probe failed: %v\n%s", err, stderr.String())
	}
	var data pobTradeData
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

// statDescriptionsPatchDrift are the descriptions the game changed in a patch after PoB-PoE2's export.
var statDescriptionsPatchDrift = map[string]bool{
	"dummy_display_defeating_arbiter_will_allow_completion_of_a_section_of_fortress": true,
}

// The stat descriptions read from go-pob-data are the pin's Data/StatDescriptions/stat_descriptions.lua, entry for
// entry.
func TestStatDescriptionsMatchPoB(t *testing.T) {
	pob := pobTradeDataRun(t)
	ours, err := StatDescriptions()
	if err != nil {
		t.Skipf("SKIP: %v", err)
	}
	if len(ours) != len(pob.StatDescriptions) {
		t.Errorf("%d descriptions, PoB-PoE2 %d", len(ours), len(pob.StatDescriptions))
	}
	bad := 0
	for i := 0; i < min(len(ours), len(pob.StatDescriptions)); i++ {
		want := pob.StatDescriptions[i]
		got := ours[i]
		same := reflect.DeepEqual(got.Stats, want.Stats) && len(got.Forms) == len(want.Forms)
		for j := 0; same && j < len(got.Forms); j++ {
			g, w := got.Forms[j], want.Forms[j]
			same = g.Text == w.Text && reflect.DeepEqual(g.Limit, w.Limit) && len(g.Flags) == len(w.Flags)
			for k := 0; same && k < len(g.Flags); k++ {
				same = g.Flags[k].K == w.Flags[k].K && g.Flags[k].V == w.Flags[k].V
			}
		}
		if !same && !statDescriptionsPatchDrift[strings.Join(want.Stats, " ")] {
			if bad++; bad <= 10 {
				t.Errorf("entry %d:\n ours     %+v\n PoB-PoE2 %+v", i+1, got, want)
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d entries differ", bad, len(pob.StatDescriptions))
	}
}

// The trade site's stats list in go-pob-data is the pin's Data/TradeSiteStats.lua, but for what the site changed
// since PoB-PoE2 fetched its copy: no id PoB-PoE2 has is gone from its category, and the entries both have keep
// PoB-PoE2's order. An id can repeat with another text. The site's additions and text changes are logged.
func TestTradeStatsMatchPoB(t *testing.T) {
	pob := pobTradeDataRun(t)
	ours, err := TradeStats()
	if err != nil {
		t.Skipf("SKIP: %v", err)
	}
	if len(ours) != len(pob.TradeStats) {
		t.Fatalf("%d categories, PoB-PoE2 %d", len(ours), len(pob.TradeStats))
	}
	for c, want := range pob.TradeStats {
		got := ours[c]
		if got.ID != want.ID || got.Label != want.Label {
			t.Errorf("category %d is %s %q, PoB-PoE2 %s %q", c, got.ID, got.Label, want.ID, want.Label)
			continue
		}
		ourIDs, pobIDs, ourEntries, pobEntries := map[string]bool{}, map[string]bool{}, map[TradeStatEntry]bool{}, map[TradeStatEntry]bool{}
		for _, e := range got.Entries {
			ourIDs[e.ID], ourEntries[e] = true, true
		}
		for _, e := range want.Entries {
			pobIDs[e.ID], pobEntries[e] = true, true
		}
		var gotOrder, wantOrder []TradeStatEntry
		for _, w := range want.Entries {
			switch {
			case !ourIDs[w.ID]:
				t.Errorf("%s: %s %q is gone", want.ID, w.ID, w.Text)
			case !ourEntries[w]:
				t.Logf("%s: %s no longer reads %q", want.ID, w.ID, w.Text)
			default:
				wantOrder = append(wantOrder, w)
			}
		}
		for _, e := range got.Entries {
			if pobEntries[e] {
				gotOrder = append(gotOrder, e)
			} else {
				t.Logf("%s: the site added %s %q", want.ID, e.ID, e.Text)
			}
		}
		if !reflect.DeepEqual(gotOrder, wantOrder) {
			t.Errorf("%s: the entries are not in PoB-PoE2's order", want.ID)
		}
	}
}
