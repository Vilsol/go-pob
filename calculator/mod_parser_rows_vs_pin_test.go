package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestParserRowsAgreeWithThePinsParseMod: each fixture line beside the pin's modLib.parseMod result (whole when it
// returned no remainder string) and its rendered mods.
//
//pob:src/Modules/ModParser.lua:7146-7161
func TestParserRowsAgreeWithThePinsParseMod(t *testing.T) {
	requireGameData(t, "3.30")

	raw, err := os.ReadFile(filepath.Join("testdata", "modparser_rows_vs_pin.tsv"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	n := 0
	for _, ln := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		parts := strings.SplitN(ln, "\t", 3)
		if len(parts) != 3 || (parts[1] != "whole" && parts[1] != "partial") {
			t.Fatalf("REFUSING: malformed fixture row %q", ln)
		}
		line, whole, want := parts[0], parts[1] == "whole", parts[2]
		n++
		e := ParseMod(line, false)
		var ours []mod.Mod
		var extra string
		if e != nil {
			ours, extra = e.ModList, e.Extra
		}
		var rendered []mcdMod
		for _, m := range ours {
			rendered = append(rendered, mcdOurMod(m))
		}
		got := mcdField(rendered, mcdMod.full)
		oursWhole := extra == ""
		if got != want || oursWhole != whole {
			t.Errorf("%q\n    pob:  whole=%v %s\n    ours: whole=%v %s (extra %q)", line, whole, want, oursWhole, got, extra)
		}
	}
	if n == 0 {
		t.Fatal("REFUSING: empty fixture")
	}
}

// TestRadiusAlsoGrantTrapsMinesAndUnarmedSkipClassStart pins the ClassStart half of ModParser.lua:6390 and 6398.
func TestRadiusAlsoGrantTrapsMinesAndUnarmedSkipClassStart(t *testing.T) {
	requireGameData(t, "3.30")

	for _, c := range []struct{ line, stat string }{
		{"Passive Skills in Radius also grant: Traps and Mines deal 3 to 5 added Physical Damage", "PhysicalMin"},
		{"Passive Skills in Radius also grant: 10% increased Unarmed Attack Speed with Melee Skills", "Speed"},
	} {
		fn := jewelFuncFor(t, c.line)
		for nodeType, wantAny := range map[string]bool{"Normal": true, "Keystone": false, "ClassStart": false} {
			out := moddb.NewModList()
			fn(&radiusJewelNode{id: "n", nodeType: nodeType, modList: moddb.NewModList()}, out, newRadiusJewelData("Tree:1"))
			got := false
			for _, m := range out.Mods() {
				got = got || m.Name() == c.stat
			}
			if got != wantAny {
				t.Errorf("%s on a %s node: granted=%v, want %v", c.stat, nodeType, got, wantAny)
			}
		}
	}
}

// TestGlobalGemPropertyRowsCarryKeyOfScaledMod pins keyOfScaledMod = "value" at ModParser.lua:3325 and 3331.
func TestGlobalGemPropertyRowsCarryKeyOfScaledMod(t *testing.T) {
	requireGameData(t, "3.30")

	for _, line := range []string{"+1 to Level of all Fireball Gems", "+1 to Level of all Fire Spell Skill Gems"} {
		e := ParseMod(line, false)
		if e == nil || len(e.ModList) != 1 || strings.TrimSpace(e.Extra) != "" {
			t.Fatalf("REFUSING: %q did not parse to one mod", line)
		}
		gp, ok := e.ModList[0].Value().List().(mod.GemProperty)
		if !ok {
			t.Fatalf("%q: value is %T, want mod.GemProperty", line, e.ModList[0].Value().List())
		}
		if gp.KeyOfScaledMod != "value" {
			t.Errorf("%q: KeyOfScaledMod = %q, want \"value\"", line, gp.KeyOfScaledMod)
		}
	}
}

// TestRegenerateEnergyShieldPerIntelligenceIsAJewelFunc pins the lower-case jewelSelfFuncs key at ModParser.lua:6530.
func TestRegenerateEnergyShieldPerIntelligenceIsAJewelFunc(t *testing.T) {
	requireGameData(t, "3.30")

	fn := jewelFuncFor(t, "Regenerate 0.4% of Energy Shield per second for every 10 Intelligence on Allocated Passives in Radius")
	d := newRadiusJewelData("Tree:1")
	work := moddb.NewModList()
	work.AddMod(mod.NewFloat("Int", mod.TypeBase, 100))
	fn(&radiusJewelNode{id: "n", nodeType: "Normal", modList: moddb.NewModList()}, work, d)
	out := moddb.NewModList()
	fn(nil, out, d)
	if got := out.Sum(mod.TypeBase, nil, "EnergyShieldRegenPercent"); got != 4 {
		t.Errorf("EnergyShieldRegenPercent = %v, want 4 (100 Int at 0.4%% per 10)", got)
	}
}
