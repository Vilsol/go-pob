package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestCalcDamageMatchesThePin pins CalcOffence.lua:110-119 and :175-176.
//
//	v2.18.1:106  local more = m_floor(skillModList:More(cfg, unpack(modNames)) * 100 + 0.50000001) / 100
//	pin:113      local more = skillModList:More(cfg, unpack(modNames))
func TestCalcDamageMatchesThePin(t *testing.T) {
	t.Run("the generic min/max attack damage rows parse", func(t *testing.T) {
		requireGameData(t, "3.29")

		for text, want := range map[string]string{
			"20% more Minimum Attack Damage": "MinDamage",
			"20% more Maximum Attack Damage": "MaxDamage",
		} {
			e := ParseMod(text, false)
			if e == nil || strings.TrimSpace(modExtra(e)) != "" {
				t.Errorf("%q does not parse cleanly", text)
				continue
			}
			got := make([]string, 0, len(e.ModList))
			for _, m := range e.ModList {
				got = append(got, m.Name())
			}
			if !contains(got, want) {
				t.Errorf("%q produced %v, want %s", text, got, want)
			}
		}

		e := ParseMod("20% more Minimum Physical Attack Damage", false)
		if e == nil {
			t.Fatal("REFUSING: the physical control does not parse; the table is broken generally")
		}
		got := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			got = append(got, m.Name())
		}
		if !contains(got, "MinPhysicalDamage") {
			t.Errorf("the physical control produced %v, want MinPhysicalDamage", got)
		}
	})

	t.Run("Volatility's generic MORE reaches the damage expression", func(t *testing.T) {
		assertBuildStats(t, "0985-gladiator-lacerate-of-haemorrhage-2.xml",
			[]string{"CombinedDPS", "BleedDPS", "TotalDotDPS"},
			func(t *testing.T, file string) {
				if !strings.Contains(file, "SupportVolatility") {
					t.Fatalf("REFUSING: the fixture no longer sockets Volatility, so it cannot " +
						"exercise a generic MinDamage/MaxDamage MORE")
				}
			})
	})

	t.Run("the damage MORE is not snapped to two decimals", func(t *testing.T) {
		assertBuildStats(t, "0030-saboteur-cyclone-2.xml",
			[]string{"IgniteFireMin", "IgniteFireMax", "IgniteTotalMin", "IgniteTotalMax"},
			nil)
	})
}

func assertBuildStats(t *testing.T, name string, stats []string, guard func(*testing.T, string)) {
	t.Helper()
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if guard != nil {
		guard(t, string(file))
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range stats {
		want := rec.Output[k]
		if want == 0 {
			t.Fatalf("REFUSING: PoB reports 0 for %s on %s; a 0-vs-0 agreement proves nothing", k, name)
		}
		if got := env.Player.Output[k]; math.Abs(got-want) > 0.005*math.Abs(want)+1e-9 {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}

// TestVolatilityStatMapStillProducesGenericMore refuses if the producer this consumer was written for ever disappears from the emitted tables.
func TestVolatilityStatMapStillProducesGenericMore(t *testing.T) {
	requireGameData(t, "3.29")

	found := map[string]bool{}
	for _, perSkill := range raw2.SkillStatMaps {
		for _, statMap := range perSkill {
			for _, m := range statMap.Mods {
				if m.Type() == mod.TypeMore && (m.Name() == "MinDamage" || m.Name() == "MaxDamage") {
					found[m.Name()] = true
				}
			}
		}
	}
	if !found["MinDamage"] || !found["MaxDamage"] {
		t.Errorf("no skill stat map emits a generic MinDamage/MaxDamage MORE any more (found %v); "+
			"calcDamage's genericMore factors have lost their only producer and the corpus "+
			"fixtures above are now vacuous", found)
	}
}
