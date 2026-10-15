package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

type gemTooltipExpected struct {
	CharacterLevel int     `json:"characterLevel"`
	Str            float64 `json:"str"`
	Dex            float64 `json:"dex"`
	Int            float64 `json:"int"`
	Tooltips       []struct {
		GemID          string `json:"gemId"`
		SkillID        string `json:"skillId"`
		VariantID      string `json:"variantId"`
		Level          int    `json:"level"`
		Quality        int    `json:"quality"`
		QualityTooltip bool   `json:"qualityTooltip"`
		Lines          []struct {
			Text      string `json:"text"`
			Separator bool   `json:"separator"`
		} `json:"lines"`
	} `json:"tooltips"`
}

func loadGemTooltipExpected(t *testing.T) gemTooltipExpected {
	t.Helper()
	f, err := os.Open("../testdata/oracle/gem_tooltip_expected.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var want gemTooltipExpected
	if err := json.NewDecoder(zr).Decode(&want); err != nil {
		t.Fatal(err)
	}
	if len(want.Tooltips) < 1000 {
		t.Fatalf("only %d expected tooltips", len(want.Tooltips))
	}
	return want
}

func tooltipText(lines []TooltipLine) string {
	var b strings.Builder
	for _, l := range lines {
		if l.Separator {
			b.WriteString("----\n")
		} else {
			b.WriteString(l.Text + "\n")
		}
	}
	return b.String()
}

// TestGemTooltipAgainstPoB: Produced by oracle/gem_tooltip.lua from the pinned PoB.
func TestGemTooltipAgainstPoB(t *testing.T) {
	want := loadGemTooltipExpected(t)
	ctx := ItemTooltipContext{CharacterLevel: want.CharacterLevel, ShowFlavourText: true, Stats: ItemTooltipStats{Str: want.Str, Dex: want.Dex, Int: want.Int}}
	wrong, total := map[bool]int{}, map[bool]int{}
	drifted := map[string]bool{}
	for _, w := range want.Tooltips {
		var expected strings.Builder
		for _, l := range w.Lines {
			if l.Separator {
				expected.WriteString("----\n")
			} else {
				expected.WriteString(l.Text + "\n")
			}
		}
		gem := pob.Gem{GemID: w.GemID, SkillID: w.SkillID, Level: w.Level, Quality: w.Quality, QualityID: "Default", Enabled: true, EnableGlobal1: true, EnableGlobal2: true}
		if w.VariantID != "" {
			gem.VariantID = &w.VariantID
		}
		var got string
		if w.QualityTooltip {
			gem.Level = 1
			got = tooltipText(GemQualityTooltip(gem))
		} else {
			tt := &ItemTooltip{}
			AddGemTooltip(tt, ctx, gem, nil, false)
			got = tooltipText(tt.Lines)
		}
		total[w.QualityTooltip]++
		if gemTooltipGameDrift[w.SkillID] && !w.QualityTooltip {
			drifted[w.SkillID] = drifted[w.SkillID] || got != expected.String()
			continue
		}
		if got != expected.String() {
			wrong[w.QualityTooltip]++
			if wrong[w.QualityTooltip] <= 3 {
				t.Errorf("%s level %d quality %d (quality tooltip %v):\n got:\n%s\nPoB:\n%s", w.SkillID, w.Level, w.Quality, w.QualityTooltip, got, expected.String())
			}
		}
	}
	for id := range gemTooltipGameDrift {
		if !drifted[id] {
			t.Errorf("%s now matches PoB: drop it from gemTooltipGameDrift", id)
		}
	}
	for _, q := range []bool{false, true} {
		if wrong[q] > 0 {
			t.Errorf("%s", fmt.Sprintf("quality tooltip %v: %d of %d differ", q, wrong[q], total[q]))
		}
	}
}

var gemTooltipGameDrift = map[string]bool{}

var gemTooltipKnownDiff = map[string]bool{}

// TestGemTooltipOfBuildGemsAgainstPoB: Produced by oracle/gem_tooltip_builds.lua from the pinned PoB.
func TestGemTooltipOfBuildGemsAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/oracle/gem_tooltip_builds_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type row struct {
		File  string   `json:"file"`
		Group int      `json:"group"`
		Gem   int      `json:"gem"`
		Lines []string `json:"lines"`
	}
	byFile := map[string][]row{}
	var files []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var r row
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if _, ok := byFile[r.File]; !ok {
			files = append(files, r.File)
		}
		byFile[r.File] = append(byFile[r.File], r)
	}
	if len(files) < 10 {
		t.Fatalf("only %d builds", len(files))
	}
	wrong, total := 0, 0
	for _, file := range files {
		env := sidebarEnv(t, file)
		displays := map[[2]int]GemDisplay{}
		for _, d := range env.GemDisplays() {
			displays[[2]int{d.Group, d.Gem}] = d.Display
		}
		set := env.Build.Skills.SkillSets[env.Build.ActiveSkillSetIndex()]
		ctx := ItemTooltipContext{CharacterLevel: env.Build.Build.Level, ShowFlavourText: true, Stats: env.ItemTooltipStats()}
		for _, r := range byFile[file] {
			want := r.Lines
			var display *GemDisplay
			if d, ok := displays[[2]int{r.Group, r.Gem}]; ok {
				display = &d
			}
			tt := &ItemTooltip{}
			AddGemTooltip(tt, ctx, set.Skills[r.Group].Gems[r.Gem], display, false)
			got := strings.Split(strings.TrimSuffix(tooltipText(tt.Lines), "\n"), "\n")
			total++
			if gemTooltipKnownDiff[fmt.Sprintf("%s/%d/%d", file, r.Group, r.Gem)] {
				if sameStrings(got, want) {
					t.Errorf("%s group %d gem %d now matches PoB: drop it from gemTooltipKnownDiff", file, r.Group, r.Gem)
				}
				continue
			}
			if !sameStrings(got, want) {
				wrong++
				if wrong <= 5 {
					t.Errorf("%s group %d gem %d:\n got %q\nPoB %q", file, r.Group, r.Gem, got, want)
				}
			}
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d tooltips differ", wrong, total)
	}
}
