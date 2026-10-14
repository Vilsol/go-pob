package calculator

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

type pobTooltipDump struct {
	Builds []struct {
		Build     string
		Error     string
		Level     int
		ClassName string
		Stats     ItemTooltipStats
		Items     []struct {
			ID     int
			Rarity string
			Lines  []struct {
				Text string
				Sep  bool
				Font string
			}
			Effective []string
		}
	}
}

var tooltipRarityColor = map[string]string{
	"NORMAL": "^xC8C8C8", "MAGIC": "^x8888FF", "RARE": "^xFFFF77", "UNIQUE": "^xAF6025", "RELIC": "^x60C060",
}

func pobTooltipLines(lines []struct {
	Text string
	Sep  bool
	Font string
}) []string {
	var out []string
	for _, l := range lines {
		if strings.Contains(l.Text, "Tip: ") {
			break
		}
		if l.Sep {
			if len(out) > 0 && out[len(out)-1] == "--" {
				continue
			}
			out = append(out, "--")
			continue
		}
		out = append(out, l.Text)
	}
	for len(out) > 0 && out[len(out)-1] == "--" {
		out = out[:len(out)-1]
	}
	return out
}

func ourTooltipLines(t ItemTooltip, rarity string) []string {
	var out []string
	for _, l := range t.Title {
		out = append(out, tooltipRarityColor[rarity]+l)
	}
	for _, l := range t.Lines {
		if strings.Contains(l.Text, "Tip: ") {
			break
		}
		if l.Separator {
			out = append(out, "--")
		} else {
			out = append(out, l.Text)
		}
	}
	for len(out) > 0 && out[len(out)-1] == "--" {
		out = out[:len(out)-1]
	}
	return out
}

var tooltipKnownDivergences = map[string]string{}

// TestAddItemTooltipAgainstPoB grades every tooltip line, colour codes included, against the pin.
func TestAddItemTooltipAgainstPoB(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join("..", "testdata", "item-tooltip", "item_tooltip-3.29.json.gz"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var dump pobTooltipDump
	if err := json.NewDecoder(zr).Decode(&dump); err != nil {
		t.Fatal(err)
	}

	items, matched, loaded, tinctures, known, flasks, flaskMatched, skillTips := 0, 0, 0, 0, 0, 0, 0, 0
	for _, b := range dump.Builds {
		if b.Error != "" {
			t.Logf("%s: skipped, the pin could not load it: %s", b.Build, b.Error)
			continue
		}
		loaded++
		build := loadCorpusBuild(t, b.Build)
		texts := map[int]string{}
		for _, it := range build.Items.Items {
			texts[it.ID] = it.Text
		}
		var misc *MiscCalculator
		ctx := ItemTooltipContext{CharacterLevel: b.Level, ClassName: b.ClassName, ShowFlavourText: true, Stats: b.Stats}
		for _, it := range b.Items {
			items++
			text, ok := texts[it.ID]
			if !ok {
				t.Errorf("%s item %d: not in our build", b.Build, it.ID)
				continue
			}
			item := PeekItem(text)
			if item.Base != nil && item.Base.Tincture != nil {
				tinctures++
			}
			if it.Effective != nil {
				if misc == nil {
					misc = GetMiscCalculator(build)
				}
				flasks++
				got := misc.EffectiveFlaskStats(item)
				if strings.Join(got, "\n") == strings.Join(it.Effective, "\n") {
					flaskMatched++
				} else {
					t.Errorf("%s item %d effective flask stats:\n%s", b.Build, it.ID, lineDiff(it.Effective, got))
				}
			}
			want := pobTooltipLines(it.Lines)
			tip := AddItemTooltip(item, ctx)
			got := ourTooltipLines(tip, it.Rarity)
			skillTip := func(texts []string) string {
				for _, x := range texts {
					if strings.Contains(x, "Tip: Hold Shift") {
						return x
					}
				}
				return ""
			}
			var pinTexts, ourTexts []string
			for _, l := range it.Lines {
				pinTexts = append(pinTexts, l.Text)
			}
			for _, l := range tip.Lines {
				ourTexts = append(ourTexts, l.Text)
			}
			if w, g := skillTip(pinTexts), skillTip(ourTexts); w != g {
				t.Errorf("%s item %d granted-skill tip: pin %q, ours %q", b.Build, it.ID, w, g)
			} else if w != "" {
				skillTips++
			}
			same := strings.Join(got, "\n") == strings.Join(want, "\n")
			key := b.Build + "#" + strconv.Itoa(it.ID)
			if reason, ok := tooltipKnownDivergences[key]; ok {
				if same {
					t.Errorf("%s: listed as divergent (%s) but now matches; drop it from the list", key, reason)
				}
				known++
				continue
			}
			if same {
				matched++
				continue
			}
			t.Errorf("%s item %d:\n%s", b.Build, it.ID, lineDiff(want, got))
		}
	}
	t.Logf("matched %d / %d items over %d builds (%d tinctures, %d known divergences); effective flask stats %d / %d", matched, items, loaded, tinctures, known, flaskMatched, flasks)
	if flasks < 50 || flaskMatched != flasks {
		t.Errorf("effective flask stats: matched %d / %d flasks", flaskMatched, flasks)
	}
	if skillTips == 0 {
		t.Errorf("REFUSING: no granted-skill tip was graded")
	}
	if tinctures == 0 {
		t.Errorf("REFUSING: the dump holds no tincture, so ItemsTab.lua:4550-4568 went ungraded")
	}
	if loaded < 20 || matched < 500 {
		t.Fatalf("REFUSING: the dump graded only %d builds / %d items", loaded, matched)
	}
}

func lineDiff(want, got []string) string {
	var b strings.Builder
	n := max(len(want), len(got))
	for i := range n {
		w, g := "", ""
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		mark := "  "
		if w != g {
			mark = "≠ "
		}
		b.WriteString(mark + strconv.Quote(w) + "  |  " + strconv.Quote(g) + "\n")
	}
	return b.String()
}
