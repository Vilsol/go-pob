package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
)

const sidebarStatBoxExpected = "../testdata/oracle/sidebar_stat_box_expected.jsonl"

type sidebarStatBoxCase struct {
	File     string `json:"file"`
	Minion   bool   `json:"minion"`
	SkillDPS []struct {
		Name      string  `json:"name"`
		DPS       float64 `json:"dps"`
		Count     float64 `json:"count"`
		Trigger   *string `json:"trigger"`
		SkillPart *string `json:"skillPart"`
		Source    *string `json:"source"`
	} `json:"skillDPS"`
	EnergyShieldProtectsMana   bool     `json:"energyShieldProtectsMana"`
	LifeCostWarningList        []string `json:"lifeCostWarningList"`
	ManaCostWarningList        []string `json:"manaCostWarningList"`
	RageCostWarningList        []string `json:"rageCostWarningList"`
	ESCostWarningList          []string `json:"esCostWarningList"`
	LifePercentCostWarningList []string `json:"lifePercentCostWarningList"`
	ManaPercentCostWarningList []string `json:"manaPercentCostWarningList"`
	CostWarning                []string `json:"costWarning"`
	JewelLimitWarning          []string `json:"jewelLimitWarning"`
	SocketLimitWarning         []string `json:"socketLimitWarning"`
	MissingAnointWarning       []string `json:"missingAnointWarning"`
}

func loadSidebarStatBoxCases(t *testing.T) []sidebarStatBoxCase {
	t.Helper()
	path := sidebarStatBoxExpected
	if p := os.Getenv("SIDEBAR_ORACLE"); p != "" {
		path = p
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var cases []sidebarStatBoxCase
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<22), 1<<22)
	for scanner.Scan() {
		var c sidebarStatBoxCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, c)
	}
	if len(cases) == 0 {
		t.Fatal("no cases in " + path)
	}
	return cases
}

func sbDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TestSkillDPSAgainstPoB: The per-skill Full DPS list PoB's sidebar shows (Calcs.lua:176-388), entry for entry.
func TestSkillDPSAgainstPoB(t *testing.T) {
	for _, want := range loadSidebarStatBoxCases(t) {
		t.Run(want.File, func(t *testing.T) {
			got := sidebarEnv(t, want.File).Player.SkillDPS
			render := func(name string, dps, count float64, trigger, part, source string) string {
				return fmt.Sprintf("%s x%v trigger=%q part=%q source=%q", name, count, trigger, part, source)
			}
			var g, w []string
			for _, s := range got {
				g = append(g, render(s.Name, s.DPS, s.Count, s.Trigger, s.SkillPart, s.Source))
			}
			for _, s := range want.SkillDPS {
				w = append(w, render(s.Name, s.DPS, s.Count, sbDeref(s.Trigger), sbDeref(s.SkillPart), sbDeref(s.Source)))
			}
			slices.Sort(g)
			slices.Sort(w)
			if !slices.Equal(g, w) {
				t.Fatalf("entries differ\n go: %q\nPoB: %q", g, w)
			}
			dps := map[string]float64{}
			for _, s := range got {
				dps[s.Name] = s.DPS
			}
			for _, s := range want.SkillDPS {
				if !oracleClose(dps[s.Name], s.DPS, oracleDumpTolerance) {
					t.Errorf("%s dps: got %v, PoB %v", s.Name, dps[s.Name], s.DPS)
				}
			}
		})
	}
}

// TestSidebarWarningsAgainstPoB: The data behind the warnings line: Calcs.lua:432-468's cost lists and CalcSetup's itemWarnings.
func TestSidebarWarningsAgainstPoB(t *testing.T) {
	for _, want := range loadSidebarStatBoxCases(t) {
		t.Run(want.File, func(t *testing.T) {
			env := sidebarEnv(t, want.File)
			got := env.SidebarWarnings()
			var costWarning []string
			for _, k := range []string{"LifeCost", "ManaCost", "RageCost", "ESCost"} {
				if env.Player.Output[k+"Warning"] != 0 {
					costWarning = append(costWarning, k)
				}
			}
			var listed []string
			for _, k := range []string{"LifeCost", "ManaCost", "RageCost", "ESCost"} {
				if slices.Contains(got.CostWarning, k) {
					listed = append(listed, k)
				}
			}
			check := func(name string, g, w []string) {
				if !sameStrings(g, w) {
					t.Errorf("%s: got %q, PoB %q", name, g, w)
				}
			}
			if got.EnergyShieldProtectsMana != want.EnergyShieldProtectsMana {
				t.Errorf("EnergyShieldProtectsMana: got %v, PoB %v", got.EnergyShieldProtectsMana, want.EnergyShieldProtectsMana)
			}
			check("LifeCostWarningList", got.LifeCostWarningList, want.LifeCostWarningList)
			check("ManaCostWarningList", got.ManaCostWarningList, want.ManaCostWarningList)
			check("RageCostWarningList", got.RageCostWarningList, want.RageCostWarningList)
			check("ESCostWarningList", got.ESCostWarningList, want.ESCostWarningList)
			check("LifePercentCostPercentCostWarningList", got.LifePercentCostPercentCostWarningList, want.LifePercentCostWarningList)
			check("ManaPercentCostPercentCostWarningList", got.ManaPercentCostPercentCostWarningList, want.ManaPercentCostWarningList)
			check("<cost>Warning", costWarning, want.CostWarning)
			check("CostWarning", listed, want.CostWarning)
			check("jewelLimitWarning", got.JewelLimitWarning, want.JewelLimitWarning)
			check("socketLimitWarning", got.SocketLimitWarning, want.SocketLimitWarning)
			check("missingAnointWarning", got.MissingAnointWarning, want.MissingAnointWarning)
		})
	}
}
