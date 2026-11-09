package calculator2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// partyExportLines is an export's lines with each run of mod lines sorted: PoB-PoE2 lists a skill's mods
// in its stat map's pairs() order, which changes from process to process.
func partyExportLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); {
		j := i
		for j < len(lines) && strings.Count(lines[j], "|") >= 5 {
			j++
		}
		slices.Sort(lines[i:j])
		i = max(j, i+1)
	}
	return lines
}

// TestPartyExportVsPoBPoE2 turns on each giver's "Export support" and compares the ExportedBuffs the
// calculation leaves in the build with what PoB-PoE2's PartyTab:Save writes (oracle/poe2_party_export.lua).
func TestPartyExportVsPoBPoE2(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "oracle", "poe2_party_export.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle map[string][]struct {
		Name string `json:"name"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatal(err)
	}
	if len(oracle) < 3 {
		t.Fatalf("the oracle holds %d givers", len(oracle))
	}
	tree := loadTestTree(t)
	for name, want := range oracle {
		t.Run(name, func(t *testing.T) {
			xml, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			build, err := pob2.ParseBuild(xml)
			if err != nil {
				t.Fatal(err)
			}
			build.Import.ExportParty = true
			if _, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)}); err != nil {
				t.Fatal(err)
			}
			got := build.Party.ExportedBuffs
			var gotNames, wantNames []string
			for _, b := range got {
				gotNames = append(gotNames, *b.Name)
			}
			for _, b := range want {
				wantNames = append(wantNames, b.Name)
			}
			if !slices.Equal(gotNames, wantNames) {
				t.Fatalf("buffers %q, PoB-PoE2 %q", gotNames, wantNames)
			}
			for i, w := range want {
				if g := partyExportLines(got[i].Text); !slices.Equal(g, partyExportLines(w.Text)) {
					t.Errorf("%s:\n%s\nPoB-PoE2:\n%s", w.Name, got[i].Text, w.Text)
				}
			}
		})
	}
}
