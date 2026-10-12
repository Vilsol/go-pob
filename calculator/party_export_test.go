package calculator

import (
	"encoding/xml"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

func partyExportEntries(name, text string) []string {
	lines := strings.Split(strings.Trim(text, pob.NotesEdgeSpace), "\n")
	if name == "PartyMemberStats" || name == "EnemyConditions" {
		slices.Sort(lines)
		return lines
	}
	if name == "EnemyMods" {
		var out []string
		for i := 0; i < len(lines); i++ {
			if !strings.Contains(lines[i], "|") {
				out = append(out, lines[i])
			} else {
				out[len(out)-1] += "\n" + lines[i]
			}
		}
		slices.Sort(out)
		return out
	}
	block := func(cur []string) string {
		head := 0
		for head < len(cur) && !strings.Contains(cur[head], "|") {
			head++
		}
		body := slices.Clone(cur[head:])
		if n := len(body); n > 0 && body[n-1] == "---" {
			body = body[:n-1]
		}
		slices.Sort(body)
		return strings.Join(append(slices.Clone(cur[:head]), body...), "\n")
	}
	var out []string
	cur := []string{}
	for _, l := range lines {
		cur = append(cur, l)
		if l == "---" || strings.HasPrefix(l, "--- Curse Limit") || (len(cur) == 2 && strings.HasPrefix(cur[0], "--- Curse Limit")) {
			if l == "---" || len(cur) == 2 {
				out = append(out, block(cur))
				cur = []string{}
			}
		}
	}
	if len(cur) > 0 {
		out = append(out, block(cur))
	}
	slices.Sort(out)
	return out
}

func partyExportCarriers(t *testing.T) (string, []string) {
	t.Helper()
	dir := corpusDirForLoadedData(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var names []string
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `exportParty="true"`) && strings.Contains(string(raw), "<ExportedBuffs") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("REFUSING: no corpus build exports its support; the fixture set is gone")
	}
	return dir, names
}

// TestPartyExportAgainstPoB compares our export of each exporting corpus build with the <ExportedBuffs> PoB saved in it.
func TestPartyExportAgainstPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dir, names := partyExportCarriers(t)
	agree, total := map[string]int{}, map[string]int{}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := map[string]string{}
		for _, b := range build.Party.ExportedBuffs {
			want[b.Name] = b.Text
		}
		if !build.ExportParty() {
			t.Fatalf("%s: exportParty=\"true\" not read", name)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.partyExports == nil {
			t.Fatalf("%s: no export collected", name)
		}
		got := env.partyExports.texts()
		for section, w := range want {
			we, ge := partyExportEntries(section, w), partyExportEntries(section, got[section])
			for _, e := range we {
				total[section]++
				if i := slices.Index(ge, e); i >= 0 {
					agree[section]++
					ge = slices.Delete(ge, i, i+1)
				} else if os.Getenv("PARTY_EXPORT_VERBOSE") != "" {
					t.Logf("%s %s: PoB has\n%s", name, section, e)
				}
			}
			if os.Getenv("PARTY_EXPORT_VERBOSE") != "" {
				for _, e := range ge {
					t.Logf("%s %s: ours only\n%s", name, section, e)
				}
			}
		}
	}
	floor := map[string]int{"PartyMemberStats": 51, "Aura": 134, "Curse": 96, "Link Skills": 2, "EnemyConditions": 608, "EnemyMods": 175}
	for _, n := range partyExportNames {
		t.Logf("%-16s %d / %d PoB entries reproduced", n[1], agree[n[1]], total[n[1]])
		if agree[n[1]] < floor[n[1]] {
			t.Errorf("%s: %d PoB entries reproduced, below the floor %d", n[1], agree[n[1]], floor[n[1]])
		}
	}
}

// TestPartyExportRoundTrips imports a build's export into another build and reads it back through the party tab.
func TestPartyExportRoundTrips(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	dir, names := partyExportCarriers(t)
	raw, err := os.ReadFile(filepath.Join(dir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	src, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	src.Party.ExportedBuffs = []pob.PartyBuffs{}
	env := BuildOutputWithCache(&Calculator{PoB: src}, OutputModeMain, NewEnvironmentCache())
	if env == nil || len(src.Party.ExportedBuffs) == 0 {
		t.Fatalf("%s: the calculation wrote no <ExportedBuffs>", names[0])
	}
	doc, err := xml.Marshal(struct {
		XMLName xml.Name  `xml:"PathOfBuilding"`
		Party   pob.Party `xml:"Party"`
	}{Party: src.Party})
	if err != nil {
		t.Fatal(err)
	}

	dst := &pob.PathOfBuilding{}
	if err := dst.ImportParty(string(doc), "All", false); err != nil {
		t.Fatal(err)
	}
	bufs := dst.PartyBuffers()
	for _, b := range src.Party.ExportedBuffs {
		if bufs[b.Name] != b.Text {
			t.Errorf("%s: imported\n%q\nexported\n%q", b.Name, bufs[b.Name], b.Text)
		}
	}
	tab := loadPartyTab(dst.Party)
	if len(tab.skipped) != 0 {
		t.Errorf("the party tab skipped exported lines: %v", tab.skipped)
	}
	found := func(name string) *partyBuff {
		for _, g := range tab.actor.Aura.byType {
			for _, m := range []map[string]*partyBuff{g.buffs, g.vaal} {
				if b := m[name]; b != nil {
					return b
				}
			}
		}
		return nil
	}
	if len(env.partyExports.aura) <= 1 {
		t.Fatalf("%s: no aura exported, so none can come back", names[0])
	}
	for name, e := range env.partyExports.aura {
		if name == "extraAura" || strings.HasSuffix(name, "_Debuff") {
			continue
		}
		if g := found(name); g == nil {
			t.Errorf("aura %s did not arrive", name)
		} else if math.Abs(g.effectMult-e.effectMult*100) > 1e-9*math.Abs(g.effectMult) {
			t.Errorf("aura %s effectMult %v, exported %v", name, g.effectMult, e.effectMult*100)
		}
	}
}
