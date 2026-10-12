package calculator

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

const partyRule = "---------------------------"

func partyLabel(name string, entries []PartySummaryEntry, parsed bool) string {
	if !parsed {
		return ""
	}
	var main, extra, other []string
	for _, e := range entries {
		switch e.Group {
		case "extraAuras":
			extra = append(extra, "  "+e.Name+": "+e.Value+"\n")
		case "otherEffects":
			other = append(other, "  "+e.Name+": "+e.Value+"\n")
		default:
			main = append(main, e.Name+": "+e.Value+"%\n")
		}
	}
	switch name {
	case "Aura":
		s := "^7" + partyRule + "\n"
		if len(main) > 0 {
			s += strings.Join(main, "") + partyRule + "\n"
		}
		if len(extra) > 0 {
			s += "extraAuras:\n" + strings.Join(extra, "") + partyRule + "\n"
		}
		if len(other) > 0 {
			s += "otherEffects:\n" + strings.Join(other, "") + partyRule + "\n"
		}
		return s
	case "EnemyMods":
		s := "^7" + partyRule
		for _, e := range entries {
			s += "\n" + e.Name + ":" + e.Value
		}
		if len(entries) > 0 {
			return s + "\n" + partyRule + "\n"
		}
		return s + "\n"
	}
	if len(main) == 0 {
		return ""
	}
	return "^7" + partyRule + "\n" + strings.Join(main, "") + partyRule + "\n"
}

func sortedLines(s string) string {
	lines := strings.Split(s, "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// TestPartySummariesMatchPoB grades the Party page's simple view against the labels PoB's ParseBuffs writes for corpus builds' exported support.
func TestPartySummariesMatchPoB(t *testing.T) {
	raw, err := os.ReadFile("../testdata/oracle/party_tab_expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Summaries []struct {
			File    string
			Buffers map[string]string
			Labels  map[string]string
		}
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want.Summaries) == 0 {
		t.Fatal("no summaries")
	}
	for _, s := range want.Summaries {
		var party pob.Party
		for _, name := range pob.PartyBufferNames {
			if s.Buffers[name] != "" {
				party.ImportedBuffs = append(party.ImportedBuffs, pob.PartyBuffs{Name: name, Text: s.Buffers[name]})
			}
		}
		got := PartySummaries(party)
		for name, w := range s.Labels {
			g := partyLabel(name, got[name], s.Buffers[name] != "")
			if name == "EnemyMods" || strings.Contains(w, "otherEffects:") {
				g, w = sortedLines(g), sortedLines(w)
			}
			if g != w {
				t.Errorf("%s %s:\n got %q\nwant %q", s.File, name, g, w)
			}
		}
	}
}
