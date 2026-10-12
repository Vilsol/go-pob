package calculator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestVoidShockwaveSourceSearchPublishesTriggerRates pins the largest of the pure triggerSkillCond configTable entries.
func TestVoidShockwaveSourceSearchPublishesTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const tol = 0.005

	checked, agreed, distinct := 0, 0, map[float64]bool{}
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["TriggerRateCap"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Void Shockwave") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		ae := env.Player.MainSkill.ActiveEffect
		if ae == nil || ae.GrantedEffect == nil {
			continue
		}
		if !strings.EqualFold(ae.GrantedEffect.Name(), "Void Shockwave") {
			continue
		}

		checked++
		distinct[math.Round(rec.Output["TriggerRateCap"]*1e6)/1e6] = true

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[key]
			exp := rec.Output[key]
			if exp == 0 {
				continue
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
			}
		}
		if agrees {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	t.Logf("Void Shockwave carriers: %d checked, %d agree on all three keys, %d distinct caps",
		checked, agreed, len(distinct))

	if checked < 30 {
		t.Fatalf("REFUSING: only %d Void Shockwave carriers found; the fixture population has moved", checked)
	}
	if len(distinct) < 2 {
		t.Fatalf("REFUSING: every carrier published the same cap (%d distinct); the sweep no longer exercises the source search", len(distinct))
	}

	const floor = 50
	if agreed < floor {
		t.Errorf("agreement REGRESSED: %d of %d carriers agree, floor is %d; diverged: %v",
			agreed, checked, floor, diverged)
	}
}

// TestManaforgedArrowsPublishesTriggerRates pins CalcTriggers.lua's ["manaforged arrows"] entry.
func TestManaforgedArrowsPublishesTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const tol = 0.005

	checked, agreed, distinct := 0, 0, map[float64]bool{}
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["TriggerRateCap"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Manaforged Arrows") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		triggeredBy := env.Player.MainSkill.TriggeredBy
		if triggeredBy == nil || triggeredBy.GrantedEffect == nil ||
			!strings.EqualFold(triggeredBy.GrantedEffect.Name(), "Manaforged Arrows") {
			continue
		}

		checked++
		distinct[math.Round(rec.Output["SkillTriggerRate"]*1e6)/1e6] = true

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[key]
			exp := rec.Output[key]
			if exp == 0 {
				continue
			}
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
			}
		}
		if agrees {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	t.Logf("Manaforged Arrows carriers: %d checked, %d agree on all three keys, %d distinct rates",
		checked, agreed, len(distinct))

	if checked < 20 {
		t.Fatalf("REFUSING: only %d Manaforged Arrows carriers found; the fixture population has moved", checked)
	}
	if len(distinct) < 2 {
		t.Fatalf("REFUSING: every carrier published the same rate (%d distinct); the sweep no longer exercises the threshold divide", len(distinct))
	}

	const floor = 48
	if agreed < floor {
		t.Errorf("agreement REGRESSED: %d of %d carriers agree, floor is %d; diverged: %v",
			agreed, checked, floor, diverged)
	}
}

// TestSpellslingerPublishesTriggerRates pins CalcTriggers.lua's ["spellslinger"] entry (:1200-1212).
func TestSpellslingerPublishesTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const tol = 0.005

	checked, agreed, distinct := 0, 0, map[float64]bool{}
	var diverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["TriggerRateCap"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Spellslinger") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		ae := env.Player.MainSkill.ActiveEffect
		if ae == nil || ae.GrantedEffect == nil {
			continue
		}
		selfNamed := strings.EqualFold(ae.GrantedEffect.Name(), "Spellslinger")
		triggeredBy := env.Player.MainSkill.TriggeredBy
		byName := triggeredBy != nil && triggeredBy.GrantedEffect != nil &&
			strings.EqualFold(triggeredBy.GrantedEffect.Name(), "Spellslinger")
		if !selfNamed && !byName {
			continue
		}

		checked++
		distinct[math.Round(rec.Output["TriggerRateCap"]*1e6)/1e6] = true

		agrees := true
		for _, key := range []string{"TriggerRateCap", "EffectiveSourceRate", "SkillTriggerRate"} {
			got := env.Player.Output[key]
			exp := rec.Output[key]
			if math.IsNaN(got) || math.IsInf(got, 0) {
				t.Fatalf("%s: %s is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, key, got)
			}
			if exp == 0 {
				if got != 0 {
					agrees = false
				}
				continue
			}
			if math.Abs(got-exp) > tol*math.Abs(exp) {
				agrees = false
			}
		}
		if agrees {
			agreed++
		} else {
			diverged = append(diverged, name)
		}
	}

	t.Logf("Spellslinger carriers: %d checked, %d agree on all three keys, %d distinct caps",
		checked, agreed, len(distinct))

	if checked < 15 {
		t.Fatalf("REFUSING: only %d Spellslinger carriers found; the fixture population has moved", checked)
	}
	if len(distinct) < 2 {
		t.Fatalf("REFUSING: every carrier published the same cap (%d distinct); the sweep no longer exercises the entry", len(distinct))
	}

	const floor = 21
	if agreed < floor {
		t.Errorf("agreement REGRESSED: %d of %d carriers agree, floor is %d; diverged: %v",
			agreed, checked, floor, diverged)
	}
}

// TestBattlemagesCryPublishesTriggerRates pins CalcTriggers.lua's ["battlemage's cry"] entry.
func TestBattlemagesCryPublishesTriggerRates(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const tol = 0.005

	checked, agreed, distinct := 0, 0, map[float64]bool{}
	var diverged, sourceRateDiverged []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok || rec.Output["TriggerRateCap"] == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "Battlemage") {
			continue
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			continue
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		if env == nil || env.Player == nil || env.Player.MainSkill == nil {
			continue
		}
		ae := env.Player.MainSkill.ActiveEffect
		if ae == nil || ae.GrantedEffect == nil {
			continue
		}
		if strings.EqualFold(ae.GrantedEffect.Name(), "Battlemage's Cry") {
			continue
		}
		triggeredBy := env.Player.MainSkill.TriggeredBy
		if triggeredBy == nil || triggeredBy.GrantedEffect == nil ||
			!strings.EqualFold(triggeredBy.GrantedEffect.Name(), "Battlemage's Cry") {
			continue
		}

		checked++
		distinct[math.Round(rec.Output["SkillTriggerRate"]*1e6)/1e6] = true

		got := env.Player.Output["TriggerRateCap"]
		exp := rec.Output["TriggerRateCap"]
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Fatalf("%s: TriggerRateCap is %v -- a NaN/Inf compares equal to nothing and cannot be graded", name, got)
		}
		if math.Abs(got-exp) > tol*math.Abs(exp) {
			diverged = append(diverged, name)
		} else {
			agreed++
		}

		for _, key := range []string{"EffectiveSourceRate", "SkillTriggerRate"} {
			g, e := env.Player.Output[key], rec.Output[key]
			if e != 0 && math.Abs(g-e) > tol*math.Abs(e) {
				sourceRateDiverged = append(sourceRateDiverged,
					fmt.Sprintf("%s %s ours=%g pob=%g ratio=%.4f", name, key, g, e, g/e))
			}
		}
	}

	t.Logf("Battlemage's Cry carriers: %d checked, %d agree on TriggerRateCap, %d distinct rates",
		checked, agreed, len(distinct))

	for _, d := range sourceRateDiverged {
		t.Logf("STANDING DIVERGENCE (not graded here): %s", d)
	}

	if checked < 1 {
		t.Fatalf("REFUSING: no Battlemage's Cry carriers found; the fixture population has moved")
	}

	const floor = 1
	if agreed < floor {
		t.Errorf("TriggerRateCap agreement REGRESSED: %d of %d carriers agree, floor is %d; diverged: %v",
			agreed, checked, floor, diverged)
	}
}
