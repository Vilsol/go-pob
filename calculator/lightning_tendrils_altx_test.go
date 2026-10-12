package calculator

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestLightningTendrilsOfEccentricityAveragesTheStrongerPulse pins act_int.lua:12642-12662 -- both of Lightning Tendrils of Eccentricity's callbacks.
func TestLightningTendrilsOfEccentricityAveragesTheStrongerPulse(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		t.Fatalf("%v", err)
	}

	const agreeFloor = 23

	carriers, agreed := 0, 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		rec, ok := oracle[name]
		if !ok {
			continue
		}
		want, published := rec.Output["AverageDamage"]
		if !published || want == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "LightningTendrilsAltX") {
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
		ms := env.Player.MainSkill
		if ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil || ms.ActiveEffect.GrantedEffect.Raw == nil ||
			ms.ActiveEffect.GrantedEffect.Raw.ID != "LightningTendrilsAltX" {
			continue
		}
		carriers++
		got := env.Player.Output["AverageDamage"]
		if math.Abs(got-want) <= math.Abs(want)*oracleDumpTolerance {
			agreed++
		}
	}

	if carriers < 25 {
		t.Fatalf("REFUSING: only %d builds run LightningTendrilsAltX as the main skill; the population has moved", carriers)
	}
	if agreed < agreeFloor {
		t.Errorf("AverageDamage agreement on Lightning Tendrils of Eccentricity is %d of %d, floor %d", agreed, carriers, agreeFloor)
	}
	t.Logf("AverageDamage on Lightning Tendrils of Eccentricity: agreed %d of %d", agreed, carriers)
}
