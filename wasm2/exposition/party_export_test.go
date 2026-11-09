package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// partyExportLines is an export's lines with each run of mod lines sorted, as PoB-PoE2 lists a skill's mods in
// pairs() order.
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

// Export Support makes the build's calculation write what PoB-PoE2's party tab exports
// (oracle/poe2_party_export.lua); the saved build carries it, and another build's party page imports it.
func TestPartyExportRoundTrips(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_party_export.json"))
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
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "party_giver.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if calc.ExportParty() || len(calc.PartyExports()) != 0 {
		t.Fatalf("Export Support is off, yet the build exports %q", calc.PartyExports())
	}
	calc.SetExportParty(true)
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if _, err := calc.PartyImpact([]string{"Life"}); err != nil {
		t.Fatal(err)
	}
	want := oracle["party_giver.xml"]
	got := calc.PartyExports()
	if len(got) != len(want) {
		t.Fatalf("exports %q, PoB-PoE2 %d", got, len(want))
	}
	for _, w := range want {
		if !slices.Equal(partyExportLines(got[w.Name]), partyExportLines(w.Text)) {
			t.Errorf("%s:\n%s\nPoB-PoE2:\n%s", w.Name, got[w.Name], w.Text)
		}
	}

	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved, `exportParty="true"`) || !strings.Contains(saved, `<ExportedBuffs name="Aura">`) {
		t.Fatalf("the saved build has no export:\n%s", saved)
	}
	member, err := ParseBuild(partyHostBuild)
	if err != nil {
		t.Fatal(err)
	}
	if err := member.AddPartyMember(saved, "All", false); err != nil {
		t.Fatal(err)
	}
	bufs := member.PartyBuffers()
	for name, text := range got {
		if bufs[name] != text {
			t.Errorf("%s imported as %q, exported %q", name, bufs[name], text)
		}
	}

	calc.SetExportParty(false)
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if len(calc.PartyExports()) != 0 {
		t.Errorf("Export Support is off again, yet the build exports %q", calc.PartyExports())
	}
}
