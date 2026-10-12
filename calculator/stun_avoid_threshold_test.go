package calculator

import (
	"fmt"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestStunAvoidAndThresholdPinCalcDefence2128 pins the three mechanisms this port dropped from CalcDefence.lua:2128-2140, all inside eight lines of one block.
//
//	if modDB:Flag(nil, "AddESToStunThreshold") then
//	    local ESMult = modDB:Sum("BASE", nil, "ESToStunThresholdPercent")
//	end
//	local notAvoidChance = modDB:Flag(nil, "StunImmune") and 0
//	    or 100 - m_min(modDB:Sum("BASE", nil, "AvoidStun"), 100)
//	    and not env.modDB:Flag(nil, "EnergyShieldProtectsMana") then
//	end
func TestStunAvoidAndThresholdPinCalcDefence2128(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var addESSeen, addESWrong int
	var immuneSeen, immuneWrong int
	var protectsSeen, protectsWrong int
	var minionSeen, minionWrong int
	var minionNames []string
	var builds, gated int
	var immuneNames []string

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		db := env.Player.ModDB
		out := env.Player.Output
		builds++

		inputsAgree := true
		for _, pool := range []string{"Life", "EnergyShield", "Mana", "totalTakenHit"} {
			if want, has := rec.Output[pool]; has && !oracleClose(out[pool], want, oracleDumpTolerance) {
				inputsAgree = false
				break
			}
		}
		if !inputsAgree {
			gated++
		}

		if db.Flag(nil, "AddESToStunThreshold") && inputsAgree {
			addESSeen++
			if want, has := rec.Output["StunThreshold"]; has &&
				!oracleClose(out["StunThreshold"], want, oracleDumpTolerance) {
				addESWrong++
			}
		}
		if db.Flag(nil, "StunImmune") && inputsAgree {
			immuneSeen++
			if want, has := rec.Output["StunAvoidChance"]; has &&
				!oracleClose(out["StunAvoidChance"], want, oracleDumpTolerance) {
				immuneWrong++
				immuneNames = append(immuneNames, fmt.Sprintf("%s(got %.4f want %.4f)", name, out["StunAvoidChance"], want))
			}
		}
		if db.Flag(nil, "EnergyShieldProtectsMana") && out["EnergyShield"] > out["totalTakenHit"] && inputsAgree {
			protectsSeen++
			if want, has := rec.Output["StunAvoidChance"]; has &&
				!oracleClose(out["StunAvoidChance"], want, oracleDumpTolerance) {
				protectsWrong++
			}
		}

		if env.Minion != nil && env.Minion.Actor != nil && db.Flag(nil, "EnergyShieldProtectsMana") {
			mOut := env.Minion.Output
			mInputsAgree := true
			for _, pool := range []string{"EnergyShield", "totalTakenHit"} {
				if want, has := rec.Minion[pool]; has && !oracleClose(mOut[pool], want, oracleDumpTolerance) {
					mInputsAgree = false
					break
				}
			}
			if mInputsAgree && mOut["EnergyShield"] > mOut["totalTakenHit"] {
				minionSeen++
				if want, has := rec.Minion["StunAvoidChance"]; has &&
					!oracleClose(mOut["StunAvoidChance"], want, oracleDumpTolerance) {
					minionWrong++
					if len(minionNames) < 5 {
						minionNames = append(minionNames, fmt.Sprintf("%s(got %.4f want %.4f)",
							name, mOut["StunAvoidChance"], want))
					}
				}
			}
		}

		if addESSeen >= 40 && immuneSeen >= 40 && protectsSeen >= 40 && minionSeen >= 40 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build was built; this test measured nothing")
	}
	t.Logf("built %d graded builds; %d gated out on a disagreeing Life/EnergyShield/Mana/totalTakenHit input", builds, gated)
	if addESSeen == 0 {
		t.Fatalf("REFUSING: over %d builds not one carries AddESToStunThreshold, so that arm "+
			"decides nothing and this test no longer measures its mechanism", builds)
	}
	if immuneSeen == 0 {
		t.Fatalf("REFUSING: over %d builds not one carries StunImmune, so the short-circuit "+
			"decides nothing and this test no longer measures its mechanism", builds)
	}
	if protectsSeen == 0 {
		t.Fatalf("REFUSING: over %d builds not one has EnergyShieldProtectsMana with "+
			"EnergyShield > totalTakenHit, so the dropped conjunct decides nothing here", builds)
	}

	if addESWrong > 0 {
		t.Errorf("AddESToStunThreshold: StunThreshold disagrees with PoB on %d of %d builds "+
			"carrying the flag", addESWrong, addESSeen)
	}
	if immuneWrong > 0 {
		t.Errorf("StunImmune: StunAvoidChance disagrees with PoB on %d of %d builds carrying "+
			"the flag: %v", immuneWrong, immuneSeen, immuneNames)
	}
	if minionSeen == 0 {
		t.Fatalf("REFUSING: over %d builds not one has a MINION with the player carrying "+
			"EnergyShieldProtectsMana and minion EnergyShield > totalTakenHit, so the actor "+
			"the conjunct reads decides nothing here", builds)
	}

	if protectsWrong > 0 {
		t.Errorf("EnergyShieldProtectsMana conjunct: StunAvoidChance disagrees with PoB on %d "+
			"of %d builds where the halving would otherwise fire", protectsWrong, protectsSeen)
	}
	if minionWrong > 0 {
		t.Errorf("EnergyShieldProtectsMana is read off the ACTOR rather than env.modDB: "+
			"Minion.StunAvoidChance disagrees with PoB on %d of %d builds where the minion's "+
			"halving would otherwise fire: %v", minionWrong, minionSeen, minionNames)
	}
	_ = mod.TypeBase
}
