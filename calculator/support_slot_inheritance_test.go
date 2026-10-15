package calculator

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestSupportSlotInheritance: CalcSetup.lua:1692-1709 — a socket group with a `source`.
//
//pob:block:src/Modules/CalcSetup.lua:1795-1812
func TestSupportSlotInheritance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, tc := range []struct {
		build             string
		wantLimit         float64
		congregationLevel int
	}{
		{"0277-reliquarian-blade-blast-2.xml", 57, 1},
		{"0409-reliquarian-desecrate.xml", 62, 4},
	} {
		t.Run(tc.build, func(t *testing.T) {
			runSupportSlotInheritanceCase(t, corpusDir, tc.build, tc.wantLimit, tc.congregationLevel)
		})
	}
}

func runSupportSlotInheritanceCase(t *testing.T, corpusDir, buildName string, wantLimit float64, wantCongregationLevel int) {
	t.Helper()
	file, err := os.ReadFile(filepath.Join(corpusDir, buildName))
	if err != nil {
		t.Fatalf("%v", err)
	}
	parsed, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: parsed}, OutputModeMain, NewEnvironmentCache())

	var phantasm *ActiveSkill
	for _, as := range env.Player.ActiveSkillList {
		if as.Minion != nil && as.Minion.MinionData != nil &&
			as.Minion.MinionData.Limit == "ActivePhantasmLimit" {
			phantasm = as
			break
		}
	}
	if phantasm == nil {
		t.Fatalf("REFUSING: no active skill on %s summons a minion limited by ActivePhantasmLimit, "+
			"so this case is measuring nothing", buildName)
	}

	byID := map[string]*GemEffect{}
	names := make([]string, 0, len(phantasm.SupportList))
	for _, se := range phantasm.SupportList {
		if se == nil || se.GrantedEffect == nil || se.GrantedEffect.Raw == nil {
			continue
		}
		byID[se.GrantedEffect.Raw.ID] = se
		names = append(names, se.GrantedEffect.Raw.ID)
	}
	sort.Strings(names)

	for _, want := range []string{"SupportSummonPhantasm", "SupportCongregation"} {
		if byID[want] == nil {
			t.Errorf("supportList is missing %s. PoB's probe reports it absorbed from another "+
				"Weapon 1 group.\n  ours: %s", want, strings.Join(names, ", "))
		}
	}

	if se := byID["SupportCongregation"]; se != nil && se.Level != wantCongregationLevel {
		t.Errorf("SupportCongregation absorbed at level %d, want %d — the support is being taken "+
			"without its gem level", se.Level, wantCongregationLevel)
	}

	if got := env.Player.Output["ActivePhantasmLimit"]; math.Abs(got-wantLimit) > 1e-9 {
		t.Errorf("ActivePhantasmLimit = %v, want %v (pinned PoB, via oracle/phantasm_limit.lua "+
			"and confirmed independently by the oracle blob)", got, wantLimit)
	}
}
