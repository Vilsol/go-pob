package pob_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	rawdata "github.com/Vilsol/go-pob/data/raw"
)

const socketGroupTextExpected = "../testdata/oracle/socket_group_text_expected.jsonl"

type pastedGem struct {
	NameSpec string `json:"nameSpec"`
	Level    int    `json:"level"`
	Quality  int    `json:"quality"`
	Enabled  bool   `json:"enabled"`
	Count    int    `json:"count"`
}

type socketGroupTextCase struct {
	File   string `json:"file"`
	Groups []struct {
		Text   *string `json:"text"`
		Pasted *struct {
			Label string      `json:"label"`
			Slot  string      `json:"slot"`
			Gems  []pastedGem `json:"gems"`
		} `json:"pasted"`
	} `json:"groups"`
}

func TestSocketGroupCopyPasteAgainstPoB(t *testing.T) {
	f, err := os.Open(socketGroupTextExpected)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	cases, checked := 0, 0
	for scanner.Scan() {
		var c socketGroupTextCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		cases++
		t.Run(c.File, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "testdata", "corpus-"+rawdata.LatestVersion, c.File))
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(raw)
			if err != nil {
				t.Fatal(err)
			}
			groups := build.Skills.SkillSets[build.ActiveSkillSetIndex()].Skills
			for g, want := range c.Groups {
				if g < len(groups) && (groups[g].Source != "" || groups[g].SourceAttr != nil) {
					continue
				}
				checked++
				text := build.CopySocketGroup(g)
				wantText := ""
				if want.Text != nil {
					wantText = *want.Text
				}
				if text != wantText {
					t.Errorf("group %d copy:\n got %q\nPoB %q", g, text, wantText)
					continue
				}
				if want.Pasted == nil {
					continue
				}
				i := build.PasteSocketGroup(text)
				if i < 0 {
					t.Errorf("group %d: paste made no group", g)
					continue
				}
				set := &build.Skills.SkillSets[build.ActiveSkillSetIndex()]
				pasted := set.Skills[i]
				set.Skills = set.Skills[:i]
				if pasted.Label != want.Pasted.Label || pasted.Slot != want.Pasted.Slot || len(pasted.Gems) != len(want.Pasted.Gems) {
					t.Errorf("group %d paste: label %q slot %q %d gems, PoB %q %q %d", g, pasted.Label, pasted.Slot, len(pasted.Gems),
						want.Pasted.Label, want.Pasted.Slot, len(want.Pasted.Gems))
					continue
				}
				for k, w := range want.Pasted.Gems {
					gem := pasted.Gems[k]
					got := pastedGem{NameSpec: gem.NameSpec, Level: gem.Level, Quality: gem.Quality, Enabled: gem.Enabled, Count: *gem.Count}
					if got != w {
						t.Errorf("group %d gem %d: got %+v, PoB %+v", g, k, got, w)
					}
				}
			}
		})
	}
	if cases == 0 || checked < 100 {
		t.Fatalf("%d builds, %d groups checked in %s", cases, checked, socketGroupTextExpected)
	}
}
