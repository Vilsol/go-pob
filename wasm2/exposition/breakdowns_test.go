package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

type breakdownSectionOracle struct {
	Type    string              `json:"type"`
	Lines   []string            `json:"lines"`
	Radius  float64             `json:"radius"`
	Label   string              `json:"label"`
	Footer  string              `json:"footer"`
	Columns []CalcsBreakdownCol `json:"columns"`
	Rows    []map[string]string `json:"rows"`
}

// breakdownsPorted are breakdowns both fixtures' main calculations hold, which must be present.
var breakdownsPorted = []string{
	"Life", "Mana", "Evasion", "FireResist", "ColdResist", "LightningResist", "ChaosResist",
	"TotemFireResist", "BlockChance", "SpellBlockChance", "GemLevel", "GemQuality", "ReqStr", "FrostShieldLife",
	"preventedLifeLossTotal", "TotemDuration", "FreezeBuildup", "PinBuildup", "ExertedAttackUptimeRatio", "BoostedAttackUptimeRatio",
}

// breakdownsPortedIn are breakdowns one fixture's main calculation holds, which must be present.
var breakdownsPortedIn = map[string][]string{
	"sorceress_spark": {"Physical", "Lightning", "Cold", "Fire", "Chaos", "LightningEffMult", "ShockChance", "ShockDPS", "ShockEffectMod"},
	"warrior_dual_mace": {
		"MainHand.Physical", "MainHand.Lightning", "MainHand.Cold", "MainHand.Fire", "MainHand.Chaos",
		"OffHand.Physical", "OffHand.Lightning", "OffHand.Cold", "OffHand.Fire", "OffHand.Chaos",
		"MainHand.PhysicalEffMult", "MainHand.FireEffMult", "OffHand.PhysicalEffMult",
		"BleedDPS", "BleedRollAverage", "BleedStackPotential", "IgniteDPS", "IgniteEffMult", "IgniteRollAverage", "IgniteStackPotential",
		"PoisonDPS", "PoisonRollAverage", "PoisonStackPotential", "MainHand.BleedChance", "MainHand.BleedDPS", "MainHand.IgniteChance",
		"MainHand.IgniteDPS", "MainHand.PoisonChance", "MainHand.PoisonDPS", "OffHand.BleedChance", "OffHand.BleedDPS",
		"OffHand.PoisonChance", "OffHand.PoisonDPS",
	},
}

// Every breakdown we show is the one PoB-PoE2's CalcBreakdownControl builds over its main calculation
// (oracle/poe2_breakdowns.lua, `mise run breakdowns-poe2`), mod list sections aside.
func TestCalcsBreakdownsMatchPoB(t *testing.T) {
	for _, name := range calcsFixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "oracle", "poe2_breakdowns_"+name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var oracle map[string][]breakdownSectionOracle
			if err := json.Unmarshal(raw, &oracle); err != nil {
				t.Fatal(err)
			}
			_, calc := readCalcsFixture(t, name)
			if _, err := calc.CalcsBreakdowns(); err == nil {
				t.Error("CalcsBreakdowns before Calculate: no error")
			}
			if _, err := calc.Calculate(); err != nil {
				t.Fatal(err)
			}
			got, err := calc.CalcsBreakdowns()
			if err != nil {
				t.Fatal(err)
			}
			for _, k := range append(append([]string(nil), breakdownsPorted...), breakdownsPortedIn[name]...) {
				if _, ok := got[k]; !ok {
					t.Errorf("%s: no breakdown", k)
				}
			}
			keys := make([]string, 0, len(got))
			for k := range got {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				compareBreakdown(t, k, oracle, shownAs(got[k], false))
				if got[k].Gear {
					compareBreakdown(t, k+"#gearOnly", oracle, shownAs(got[k], true))
				}
			}
		})
	}
}

// shownAs is a breakdown's sections as a cell shows them, a gearOnly one with the gear table in place of the slots
// table.
func shownAs(bd CalcsBreakdown, gearOnly bool) CalcsBreakdown {
	out := CalcsBreakdown{Lines: bd.Lines, Radius: bd.Radius}
	for _, tb := range bd.Tables {
		if gearOnly && len(tb.Columns) > 0 && tb.Columns[0].Key == "base" && tb.Columns[0].Label == "Base" {
			tb = bd.GearTable
		}
		out.Tables = append(out.Tables, tb)
	}
	return out
}

func compareBreakdown(t *testing.T, k string, oracle map[string][]breakdownSectionOracle, got CalcsBreakdown) {
	t.Helper()
	sections, ok := oracle[k]
	if !ok {
		t.Errorf("%s: PoB-PoE2 holds no such breakdown", k)
		return
	}
	var want CalcsBreakdown
	for _, s := range sections {
		switch s.Type {
		case "TEXT":
			want.Lines = s.Lines
		case "RADIUS":
			want.Radius = s.Radius
		case "TABLE":
			rows := s.Rows
			if rows == nil {
				rows = []map[string]string{}
			}
			want.Tables = append(want.Tables, CalcsBreakdownTable{Label: s.Label, Footer: s.Footer, Columns: s.Columns, Rows: rows})
		}
	}
	if !reflect.DeepEqual(got, want) {
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		t.Errorf("%s:\n PoB-PoE2 %s\n ours     %s", k, w, g)
	}
}

// The sidebar's breakdowns are its main calculation's (Build.lua:533, PoB-PoE2), not the Calcs tab's own.
func TestSidebarBreakdownsAreTheMainCalculations(t *testing.T) {
	_, calc := readCalcsInputFixture(t, calcsInputFixtures[0])
	sidebar, err := calc.SidebarBreakdowns()
	if err != nil {
		t.Fatal(err)
	}
	calcs, err := calc.CalcsBreakdowns()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sidebar["TotalDPS"].Lines, calcBreakdownLines(calc.env, "TotalDPS")) {
		t.Errorf("sidebar TotalDPS %v, main %v", sidebar["TotalDPS"].Lines, calcBreakdownLines(calc.env, "TotalDPS"))
	}
	if reflect.DeepEqual(sidebar["TotalDPS"].Lines, calcs["TotalDPS"].Lines) {
		t.Error("the fixture's Calcs tab shows the sidebar's skill; it tests nothing")
	}
	sd := calculator2.ModSectionData{ModName: []string{"Damage"}, ModType: "INC", Cfg: "skill"}
	main, err := calc.SidebarModSection(sd)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(main, calc.env.ModSection(sd)) {
		t.Error("the sidebar's mod table is not the main calculation's")
	}
}

func calcBreakdownLines(env *calculator2.Environment, name string) []string {
	return env.Player.Breakdown.Views()[name].Lines
}
