package pob

import (
	"encoding/xml"
	"testing"
)

const (
	memberA = `<PathOfBuilding><Build level="90" className="Templar" ascendClassName="Guardian"/><Party>` +
		`<ExportedBuffs name="Aura">AuraA</ExportedBuffs><ExportedBuffs name="PartyMemberStats">StatsA</ExportedBuffs></Party></PathOfBuilding>`
	memberB = `<PathOfBuilding><Build level="85" className="Witch" ascendClassName="Necromancer"/><Party>` +
		`<ExportedBuffs name="Aura">AuraB</ExportedBuffs></Party></PathOfBuilding>`
)

// TestPartyMembersCombineActiveImports: UI_ISSUES Party: each imported build is a member; the buffers are PoB's imports of the active members, appended.
func TestPartyMembersCombineActiveImports(t *testing.T) {
	want := func(imports ...string) map[string]string {
		b := &PathOfBuilding{}
		for i, x := range imports {
			if err := b.ImportParty(x, "All", i > 0); err != nil {
				t.Fatal(err)
			}
		}
		return b.PartyBuffers()
	}
	same := func(what string, b *PathOfBuilding, w map[string]string) {
		t.Helper()
		got := b.PartyBuffers()
		for _, name := range PartyBufferNames {
			if got[name] != w[name] {
				t.Errorf("%s: %s = %q, want %q", what, name, got[name], w[name])
			}
		}
	}

	b := &PathOfBuilding{}
	if err := b.AddPartyMember(memberA, "All", false); err != nil {
		t.Fatal(err)
	}
	if err := b.AddPartyMember(memberB, "All", true); err != nil {
		t.Fatal(err)
	}
	m := b.PartyMembers()
	if len(m) != 2 || m[0].Name != "Guardian" || m[0].Level != 90 || m[1].Ascendancy != "Necromancer" || !m[0].Active || !m[1].Active {
		t.Fatalf("members: %+v", m)
	}
	same("both", b, want(memberA, memberB))

	b.SetPartyMemberActive(0, false)
	same("A off", b, want(memberB))
	if b.PartyMembers()[0].Active {
		t.Error("A still active")
	}

	out, err := xml.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var back PathOfBuilding
	if err := xml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.PartyMembers(); len(got) != 2 || got[0].Active || !got[1].Active || got[1].Name != "Necromancer" {
		t.Fatalf("members after a save and load: %+v", got)
	}
	back.SetPartyMemberActive(0, true)
	same("A on after load", &back, want(memberA, memberB))
	applied := map[string]string{}
	for _, p := range back.Party.Applied() {
		applied[p.Name] = p.Text
	}
	if applied["Aura"] != "AuraA\nAuraB" {
		t.Errorf("applied Aura = %q", applied["Aura"])
	}

	back.RemovePartyMember(1)
	same("B removed", &back, want(memberA))

	if err := back.AddPartyMember(memberB, "All", false); err != nil {
		t.Fatal(err)
	}
	if got := back.PartyMembers(); len(got) != 2 || got[0].Active || !got[1].Active {
		t.Fatalf("members after a replacing import: %+v", got)
	}
	same("replaced", &back, want(memberB))
}
