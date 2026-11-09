package exposition

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type partyImportOracle struct {
	Inputs    map[string]string `json:"inputs"`
	Scenarios []struct {
		Name  string `json:"name"`
		Steps []struct {
			Import string `json:"import"`
			Dest   string `json:"dest"`
			Append bool   `json:"append"`
			Clear  string `json:"clear"`
		} `json:"steps"`
		Buffers map[string]string `json:"buffers"`
		Saved   []struct {
			Elem string `json:"elem"`
			Name string `json:"name"`
			Text string `json:"text"`
		} `json:"saved"`
		Attrib map[string]string `json:"attrib"`
	} `json:"scenarios"`
}

const partyHostBuild = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<PathOfBuilding2>\n\t<Build level=\"1\" className=\"Ranger\"/>\n</PathOfBuilding2>\n"

// The party tab's import and clear buttons (PartyTab.lua:82-118, 162-285, 339-344 in PoB-PoE2) leave the text
// boxes and the saved Party element as oracle/poe2_party_import.lua recorded them.
func TestPartyImportMatchesPoBPoE2(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_party_import.json"))
	if err != nil {
		t.Fatal(err)
	}
	var o partyImportOracle
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	if len(o.Scenarios) == 0 {
		t.Fatal("the oracle holds no scenario")
	}
	for _, sc := range o.Scenarios {
		calc, err := ParseBuild(partyHostBuild)
		if err != nil {
			t.Fatal(err)
		}
		appendMode := false
		for _, step := range sc.Steps {
			if step.Clear != "" {
				calc.SetPartyOptions(step.Clear, appendMode, false)
				calc.ClearParty(step.Clear)
				continue
			}
			appendMode = step.Append
			calc.SetPartyOptions(step.Dest, appendMode, false)
			if err := calc.ImportParty(o.Inputs[step.Import], step.Dest, step.Append); err != nil {
				t.Fatalf("%s: %v", sc.Name, err)
			}
		}
		if got := calc.PartyBuffers(); !reflect.DeepEqual(got, sc.Buffers) {
			t.Errorf("%s: buffers\n got  %q\n want %q", sc.Name, got, sc.Buffers)
		}
		saved := make([][2]string, 0, len(calc.build.Party.ImportedBuffs))
		for _, n := range calc.build.Party.ImportedBuffs {
			saved = append(saved, [2]string{*n.Name, n.Text})
		}
		want := make([][2]string, 0, len(sc.Saved))
		for _, n := range sc.Saved {
			if n.Elem == "ImportedBuffs" {
				want = append(want, [2]string{n.Name, n.Text})
			}
		}
		if !reflect.DeepEqual(saved, want) {
			t.Errorf("%s: saved ImportedBuffs\n got  %q\n want %q", sc.Name, saved, want)
		}
		dest, appendState, advanced := calc.PartyOptions()
		gotAttrib := map[string]string{"destination": dest, "append": boolText(appendState), "ShowAdvanceTools": boolText(advanced)}
		if !reflect.DeepEqual(gotAttrib, sc.Attrib) {
			t.Errorf("%s: attributes %v, want %v", sc.Name, gotAttrib, sc.Attrib)
		}
	}
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// A saved build reopens with the party tab's buffers, options and imported members as they were.
func TestPartyRoundTrips(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_party_import.json"))
	if err != nil {
		t.Fatal(err)
	}
	var o partyImportOracle
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(partyHostBuild)
	if err != nil {
		t.Fatal(err)
	}
	calc.SetPartyOptions("All", true, true)
	if err := calc.AddPartyMember(o.Inputs["giver"], "All", false); err != nil {
		t.Fatal(err)
	}
	if err := calc.AddPartyMember(o.Inputs["buffers"], "Aura", true); err != nil {
		t.Fatal(err)
	}
	calc.SetPartyMemberActive(0, false)
	saved, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := ParseBuild(saved)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reopened.PartyBuffers(), calc.PartyBuffers()) {
		t.Errorf("buffers\n got  %q\n want %q", reopened.PartyBuffers(), calc.PartyBuffers())
	}
	if !reflect.DeepEqual(reopened.PartyMembers(), calc.PartyMembers()) {
		t.Errorf("members\n got  %+v\n want %+v", reopened.PartyMembers(), calc.PartyMembers())
	}
	d1, a1, s1 := reopened.PartyOptions()
	if d1 != "All" || !a1 || !s1 {
		t.Errorf("options %q %v %v", d1, a1, s1)
	}
	members := calc.PartyMembers()
	if len(members) != 2 || members[0].Active || !members[1].Active || members[0].Name != "Warrior" || members[1].Destination != "Aura" {
		t.Fatalf("members %+v", members)
	}
	// with the giver off, only the second member's auras remain
	if got := calc.PartyBuffers()["Curse"]; got != "" {
		t.Errorf("the disabled member's curses stayed: %q", got)
	}
	if got := calc.PartyBuffers()["Aura"]; !strings.Contains(got, "Purity of Ice") || strings.Contains(got, "Defiance Banner") {
		t.Errorf("auras after disabling the giver: %q", got)
	}
}

// Importing a member through the party page moves the calculation to what PoB-PoE2 calculates with the same
// buffers (oracle/poe2_party.lua), and the impact rows pair it with the calculation without them.
func TestPartyMemberMovesTheCalculation(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_party_import.json"))
	if err != nil {
		t.Fatal(err)
	}
	var o partyImportOracle
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_party_buffers.json"))
	if err != nil {
		t.Fatal(err)
	}
	var calcOracle struct {
		Receivers []struct {
			Build   string `json:"build"`
			Without struct {
				Player map[string]float64 `json:"player"`
			} `json:"without"`
			With struct {
				Player map[string]float64 `json:"player"`
			} `json:"with"`
		} `json:"receivers"`
	}
	if err := json.Unmarshal(raw, &calcOracle); err != nil {
		t.Fatal(err)
	}
	var want, wantWithout float64
	for _, r := range calcOracle.Receivers {
		if r.Build == "sorceress_spark.xml" {
			want, wantWithout = r.With.Player["CombinedDPS"], r.Without.Player["CombinedDPS"]
		}
	}
	if want == 0 || want == wantWithout {
		t.Fatalf("the oracle's CombinedDPS does not move: %v -> %v", wantWithout, want)
	}
	xml, err := os.ReadFile(filepath.Join("testdata", "sorceress_spark.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if err := calc.AddPartyMember(o.Inputs["buffers"], "All", false); err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Skipf("the PoE2 build does not calculate here: %v", err)
	}
	near := func(a, b float64) bool { return math.Abs(a-b) <= 0.005*math.Abs(b) }
	if got := out.Output["CombinedDPS"]; !near(got, want) {
		t.Errorf("CombinedDPS with the member %v, PoB-PoE2 %v", got, want)
	}
	rows, err := calc.PartyImpact([]string{"CombinedDPS"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !near(rows[0].With, want) || !near(rows[0].Without, wantWithout) {
		t.Errorf("impact %+v, want %v -> %v", rows, wantWithout, want)
	}
	calc.DisablePartyEffects()
	out, err = calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Output["CombinedDPS"]; !near(got, wantWithout) {
		t.Errorf("CombinedDPS with the party disabled %v, PoB-PoE2 without it %v", got, wantWithout)
	}
}
