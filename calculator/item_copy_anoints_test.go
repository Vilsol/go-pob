package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// TestCopyAnointsAndEldritchImplicitsAgainstPoB: ItemsTab.lua:1817-1869: a new item made the display item over the build's equipped gear.
func TestCopyAnointsAndEldritchImplicitsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/party-ui/item_copy_anoints.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/party-ui/item_copy_anoints.json: %v", err)
	}
	type want struct {
		Enchant   []string `json:"enchant"`
		Implicit  []string `json:"implicit"`
		Cleansing bool     `json:"cleansing"`
		Tangle    bool     `json:"tangle"`
		Quality   int      `json:"quality"`
	}
	var oracle map[string]struct {
		Candidates []string `json:"candidates"`
		Items      []want   `json:"items"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	lines := func(l []ModLine) []string {
		out := make([]string, 0, len(l))
		for _, m := range l {
			out = append(out, m.Line)
		}
		return out
	}
	copied, checked := 0, 0
	for name, entry := range oracle {
		build := loadCorpusBuild(t, name)
		for k, text := range entry.Candidates {
			item := ParseItem(text)
			CopyAnointsAndEldritchImplicits(build, item, true, false, "")
			item.NormaliseQuality()
			item = ParseItem(buildRawText(item))
			w := entry.Items[k]
			q := 0
			if item.Quality != nil {
				q = *item.Quality
			}
			got := want{lines(item.EnchantModLines), lines(item.ImplicitModLines), item.Influence.Cleansing, item.Influence.Tangle, q}
			if !slices.Equal(got.Enchant, append([]string{}, w.Enchant...)) || !slices.Equal(got.Implicit, append([]string{}, w.Implicit...)) ||
				got.Cleansing != w.Cleansing || got.Tangle != w.Tangle || got.Quality != w.Quality {
				t.Errorf("%s candidate %d: got %+v, PoB %+v", name, k+1, got, w)
			}
			if len(w.Enchant)+len(w.Implicit) > 0 {
				copied++
			}
			checked++
		}
	}
	if copied < 100 {
		t.Fatalf("vacuity: only %d of %d candidates took anything from the equipped gear", copied, checked)
	}
	t.Logf("%d candidates over %d builds, %d took something", checked, len(oracle), copied)
}

// TestNormaliseDisplayItem: ItemsTab.lua:1859-1869: the pasted text comes back with the equipped amulet's anoint and normalised quality.
func TestNormaliseDisplayItem(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0065-chieftain-tectonic-slam.xml")
	got := ParseItem(NormaliseDisplayItem(build, "Rarity: RARE\nCandidate Amulet\nOnyx Amulet\n+10 to all Attributes", true))
	if len(got.EnchantModLines) != 1 || got.EnchantModLines[0].Line != "Allocates Vigour" {
		t.Fatalf("enchants %+v", got.EnchantModLines)
	}
	if text := "not an item"; NormaliseDisplayItem(build, text, true) != text {
		t.Fatal("text with no base was rewritten")
	}
}

// TestRareLikeUniqueCustomLines: ItemsTab.lua:2344, Item.lua:1011-1016.
func TestRareLikeUniqueCustomLines(t *testing.T) {
	initItemsTabData(t)
	for _, c := range []struct {
		text string
		want bool
	}{
		{"Rarity: UNIQUE\nDread Captain's Cutlass\nGhostflame Blade\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nThe Crimson Storm\nSteelwood Bow\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nSubsume the Source\nFaithful Helmet\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nParadoxica\nVaal Rapier\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nCane of Kulemak\nSerpentine Staff\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nReplica Paradoxica\nVaal Rapier\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nThe Queen's Hunger\nVaal Regalia\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nThat Which Was Taken\nCrimson Jewel\n{custom}+10 to maximum Life", true},
		{"Rarity: UNIQUE\nGoldrim\nLeather Cap\n{custom}+10 to maximum Life", false},
	} {
		info, _ := DisplayItem(c.text)
		k := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.Section == "explicit" })
		if k < 0 || info.Mods[k].Removable != c.want {
			t.Errorf("%s: explicit controls %+v, want Remove %v", strings.Split(c.text, "\n")[1], info.Mods, c.want)
		}
	}
}
