package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// TestModBrowserSearchAgainstPoB: Produced by oracle/custom_mod_browser.lua with searches.
func TestModBrowserSearchAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile("../testdata/oracle/custom_mod_browser_search.json")
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Searches []struct {
			Search string   `json:"search"`
			Shown  []string `json:"shown"`
		} `json:"searches"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want.Searches) < 5 {
		t.Fatalf("REFUSING: %d searches", len(want.Searches))
	}
	for _, s := range want.Searches {
		got := ModBrowserList(s.Search)
		if !slices.Equal(got, s.Shown) {
			first := 0
			for first < len(got) && first < len(s.Shown) && got[first] == s.Shown[first] {
				first++
			}
			t.Errorf("search %q: %d lines, PoB %d; first difference at %d", s.Search, len(got), len(s.Shown), first)
		}
	}
}

// TestModBrowserSupported: ConfigModBrowser.lua:89-90: a line is supported when the parser reads all of it.
func TestModBrowserSupported(t *testing.T) {
	if !ModBrowserSupported("10% increased Attack Speed while Chilled") {
		t.Error("a line the parser reads")
	}
	if ModBrowserSupported("No matching modifiers found") || ModBrowserSupported("Quux the frobnicator") {
		t.Error("the empty entry, a line it cannot read")
	}
}

// TestModBrowserFuzzy: UI_ISSUES Config 1: PoB's matches first in PoB's order, then lines the query's words are a loose subsequence of.
func TestModBrowserFuzzy(t *testing.T) {
	for _, s := range []string{"life", "  Fire RES ", "attack speed while", "crit multi"} {
		pob := ModBrowserList(s)
		got := ModBrowserListFuzzy(s)
		if len(got) < len(pob) || !slices.Equal(got[:len(pob)], pob) {
			t.Errorf("search %q: PoB's %d lines are not the prefix of the fuzzy list", s, len(pob))
		}
	}
	for search, want := range map[string][]string{"incdmg": {"increase", "damage"}, "fire damge": {"fire", "damage"}, "lightnign res": {"lightning", "res"}} { //nolint:misspell // the typos are the subject
		if pob := ModBrowserList(search); !slices.Equal(pob, []string{ModBrowserEmpty}) {
			t.Fatalf("search %q: PoB finds %d lines, not a fuzzy-only case", search, len(pob))
		}
		got := ModBrowserListFuzzy(search)
		if len(got) == 0 || got[0] == ModBrowserEmpty {
			t.Errorf("search %q: no fuzzy match", search)
			continue
		}
		for _, w := range want {
			if !strings.Contains(strings.ToLower(got[0]), w) {
				t.Errorf("search %q: first match %q lacks %q", search, got[0], w)
			}
		}
	}
	if got := ModBrowserListFuzzy("zzzzq"); !slices.Equal(got, []string{ModBrowserEmpty}) {
		t.Errorf("zzzzq: %d lines, want the empty entry", len(got))
	}
}

// TestModBrowserSources: ConfigModBrowser.lua:153-250 (#10168): tree and mastery lines are listed, and a line's tooltip names its sources.
func TestModBrowserSources(t *testing.T) {
	for line, want := range map[string][]string{
		"Action Speed cannot be modified to below Base Value": {"Ascendancy Normal Node", "Ascendancy Notable Node", "Item mod"},
		"All Damage with Maces and Sceptres inflicts Chill":   {"Mastery Node"},
		"2% additional Physical Damage Reduction":             {"Crafted mod", "Item mod"},
	} {
		if got := ModBrowserSources(line); !slices.Equal(got, want) {
			t.Errorf("%q: sources %v, want %v", line, got, want)
		}
		if !slices.Contains(ModBrowserList(""), line) {
			t.Errorf("%q is not listed", line)
		}
	}
	if got := ModBrowserSources("Quux the frobnicator"); got != nil {
		t.Errorf("an unlisted line has sources %v", got)
	}
}
