package pob

import (
	"encoding/json"
	"os"
	"testing"
)

type partyOracle struct {
	Inputs    map[string]string
	Scenarios []struct {
		Name  string
		Steps []struct {
			Import string
			Dest   string
			Append bool
			Clear  *string
		}
		Buffers map[string]string
		Saved   [][3]string
		Attrib  map[string]string
	}
}

// TestPartyImportMatchesPoB replays PoB's Import / Append / Clear.
func TestPartyImportMatchesPoB(t *testing.T) {
	raw, err := os.ReadFile("../testdata/oracle/party_tab_expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var want partyOracle
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want.Scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	for _, sc := range want.Scenarios {
		b := &PathOfBuilding{}
		for _, step := range sc.Steps {
			if step.Clear != nil {
				b.SetPartyOptions(*step.Clear, b.Party.Append == "true", false)
				b.ClearParty(*step.Clear)
				continue
			}
			text, ok := want.Inputs[step.Import]
			if !ok {
				t.Fatalf("no input %s", step.Import)
			}
			b.SetPartyOptions(step.Dest, step.Append, false)
			if err := b.ImportParty(text, step.Dest, step.Append); err != nil {
				t.Fatalf("%s: %v", sc.Name, err)
			}
		}
		got := b.PartyBuffers()
		for name, w := range sc.Buffers {
			if got[name] != w {
				t.Errorf("%s: buffer %s:\n got %q\nwant %q", sc.Name, name, got[name], w)
			}
		}
		if len(b.Party.ImportedBuffs) != len(sc.Saved) {
			t.Errorf("%s: saved %d <ImportedBuffs>, PoB %d", sc.Name, len(b.Party.ImportedBuffs), len(sc.Saved))
		} else {
			for i, node := range b.Party.ImportedBuffs {
				if w := sc.Saved[i]; w[0] != "ImportedBuffs" || node.Name != w[1] || node.Text != w[2] {
					t.Errorf("%s: saved node %d is %s, PoB %s %s", sc.Name, i, node.Name, w[0], w[1])
				}
			}
		}
		attrib := map[string]string{"destination": b.Party.Destination, "append": b.Party.Append, "ShowAdvanceTools": b.Party.ShowAdvanceTools}
		for k, w := range sc.Attrib {
			if attrib[k] != w {
				t.Errorf("%s: <Party %s> %q, PoB %q", sc.Name, k, attrib[k], w)
			}
		}
	}
}

// TestPartyAppliedFollowsPoBActor pins when the calculation's party changes.
func TestPartyAppliedFollowsPoBActor(t *testing.T) {
	names := func(bufs []PartyBuffs) map[string]string {
		out := map[string]string{}
		for _, b := range bufs {
			out[b.Name] = b.Text
		}
		return out
	}
	b := &PathOfBuilding{}
	b.SetPartyBuffer("Aura", "a")
	b.SetPartyBuffer("EnemyMods", "m")
	b.SetPartyBuffer("PartyMemberStats", "s")
	if got := names(b.Party.Applied()); len(got) != 0 {
		t.Fatalf("an edit applied before Rebuild All: %v", got)
	}
	if b.PartyEffectsApplied() {
		t.Error("PartyEffectsApplied after an edit")
	}
	b.RebuildParty()
	if got := names(b.Party.Applied()); got["Aura"] != "a" || got["EnemyMods"] != "m" || !b.PartyEffectsApplied() {
		t.Errorf("Rebuild All: %v", got)
	}
	b.DisablePartyEffects()
	if got := b.Party.Applied(); len(got) != 0 || b.PartyBuffers()["Aura"] != "a" {
		t.Errorf("Disable Party Effects: applied %v, buffers %v", got, b.PartyBuffers())
	}
	b.SetPartyBuffer("Curse", "c")
	if !b.PartyEffectsDisabled() || len(b.Party.Applied()) != 0 {
		t.Error("an edit re-enabled the party effects")
	}
	b.RebuildParty()
	if b.PartyEffectsDisabled() {
		t.Error("PartyEffectsDisabled after Rebuild All")
	}
	if c := b.WithoutPartyEffects(); !c.PartyEffectsDisabled() || b.PartyEffectsDisabled() {
		t.Error("WithoutPartyEffects must disable the copy only")
	}
	b.ClearParty("Aura")
	got := names(b.Party.Applied())
	if got["Aura"] != "" || got["EnemyMods"] != "" || got["PartyMemberStats"] != "s" {
		t.Errorf("Clear Aura: applied %v", got)
	}
	if bufs := b.PartyBuffers(); bufs["EnemyMods"] != "m" || bufs["Aura"] != "" {
		t.Errorf("Clear Aura: buffers %v", bufs)
	}
}
