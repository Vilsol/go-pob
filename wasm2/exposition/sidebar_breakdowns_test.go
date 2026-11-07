package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type sidebarBreakdownOracle struct {
	Index     int      `json:"index"`
	Stat      string   `json:"stat"`
	ChildStat string   `json:"childStat"`
	Breakdown *string  `json:"breakdown"`
	ModNames  []string `json:"modNames"`
	Key       string   `json:"key"`
	Source    string   `json:"source"`
}

// The breakdown a sidebar stat names is the one PoB-PoE2's GetStatBreakdownKey gives over its main calculation
// (oracle/poe2_sidebar_breakdowns.lua, `mise run breakdowns-poe2`), but for a breakdown we do not write yet.
func TestSidebarBreakdownKeysMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_sidebar_breakdowns_"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var oracle []sidebarBreakdownOracle
			if err := json.Unmarshal(raw, &oracle); err != nil {
				t.Fatal(err)
			}
			_, calc := readCalcsFixture(t, name)
			if _, err := calc.SidebarBreakdownKey("Life", "", "", false, false); err == nil {
				t.Error("SidebarBreakdownKey before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			bd := calc.env.Player.Breakdown
			matched, unwritten := 0, 0
			for _, o := range oracle {
				override := ""
				if o.Breakdown != nil {
					override = *o.Breakdown
				}
				ref, err := calc.SidebarBreakdownKey(o.Stat, o.ChildStat, override, o.Breakdown != nil, o.ModNames != nil)
				if err != nil {
					t.Fatal(err)
				}
				got := ref.Key
				wantSource := o.Source
				if wantSource == "" {
					wantSource = o.Key
				}
				switch {
				case got == o.Key && ref.Source == wantSource:
					matched++
				case got == "" && bd.Entries[o.Key] == nil && bd.Entries[o.Stat] == nil:
					unwritten++
				default:
					t.Errorf("display stat %d (%s): PoB-PoE2 %q from %q, ours %q from %q", o.Index, o.Stat, o.Key, wantSource, got, ref.Source)
				}
			}
			t.Logf("%d / %d keys as PoB-PoE2 gives them, %d breakdowns not written", matched, len(oracle), unwritten)
			if matched < 30 {
				t.Errorf("only %d keys matched", matched)
			}
		})
	}
}
