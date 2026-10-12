package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestAutoexertionSkillsReachTheReservationGate pins the `or` arm of CalcPerform.lua:1940.
//
//	and not activeSkill.skillTypes[SkillType.ReservationBecomesCost] then
func TestAutoexertionSkillsReachTheReservationGate(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	_, names := corpusNamesForLoadedData(t)

	const scanCap = 300
	var carriers, gatedOutBefore, scanned int
	for _, name := range names {
		if scanned >= scanCap {
			break
		}
		scanned++
		env := corpusEnvForLoadedData(t, name)
		if env == nil || env.Player == nil {
			continue
		}
		for _, as := range env.Player.ActiveSkillList {
			if as.SkillData == nil || !as.SkillData.TriggeredByAutoexertion {
				continue
			}
			carriers++
			if !as.SkillTypes[data.SkillTypeHasReservation] {
				gatedOutBefore++
			}
		}
		if carriers > 0 && gatedOutBefore > 0 {
			break
		}
	}

	if carriers == 0 {
		t.Fatalf("REFUSING: no skill with TriggeredByAutoexertion in the first %d corpus builds, so "+
			"this test cannot distinguish a working disjunct from an unreachable one. At a ~7.4%% "+
			"base rate that is itself a finding: either the producer regressed "+
			"(skill_stat_map.go:306) or the corpus no longer covers it.", scanned)
	}
	t.Logf("Autoexertion-triggered skills seen: %d; of those, %d do NOT declare HasReservation "+
		"and were excluded from the reservation loop before this arm existed.", carriers, gatedOutBefore)
	if gatedOutBefore != 0 {
		t.Errorf("%d of %d Autoexertion-triggered skill(s) do NOT declare HasReservation, so the "+
			"`or` arm at CalcPerform.lua:1940 is now LOAD-BEARING where it was inert when ported. "+
			"That is not a failure of the port -- it means this population changed and the "+
			"reservation numbers for those skills moved. Re-measure the gate and update this note.",
			gatedOutBefore, carriers)
	}
}
