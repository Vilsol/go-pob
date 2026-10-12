package calculator

import (
	"fmt"
	"strings"
	"testing"
)

// TestPactUptimeBlock pins CalcOffence.lua:2804-2851.
//
//	globalOutput["PactOf"..key.."Cooldown"]  ["PactOf"..key.."CastTime"]
//	globalOutput[key.."EmpoweredCount"]      [key.."UpTimeRatio"]
//	skillModList:NewMod(...) scaled by effectMult   -- :2845-2849
func TestPactUptimeBlock(t *testing.T) {
	t.Run("Lycia's uptime, cooldown and cast time", func(t *testing.T) {
		assertBuildStats(t, "0207-occultist-winter-orb.xml", []string{
			"LyciaEmpoweredCount", "PactOfLyciaCooldown", "PactOfLyciaCastTime", "LyciaUpTimeRatio",
		}, nil)
	})

	t.Run("Beidat's uptime and cooldown", func(t *testing.T) {
		assertBuildStats(t, "0182-assassin-freezing-pulse.xml", []string{
			"BeidatEmpoweredCount", "BeidatUpTimeRatio", "PactOfBeidatCooldown", "PactOfBeidatCastTime",
		}, nil)
	})
}

// TestBeidatBeamChainsString pins CalcOffence.lua:2857-2861: the Beidat beam-chain bonus rewrites ChainMaxString.
func TestBeidatBeamChainsString(t *testing.T) {
	for _, name := range []string{"0217-elementalist-arc-of-oscillating.xml", "0864-reliquarian-dark-bargain-2.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"ChainMax", "ChainMaxString"}, nil)
		})
	}
}

// TestBattlemageCryBlock pins CalcOffence.lua:2716-2741.
func TestBattlemageCryBlock(t *testing.T) {
	assertBuildStats(t, "0113-chieftain-ice-crash.xml", []string{
		"BattleMageCryCooldown", "BattleMageCryDuration", "BattlemageUpTimeRatio", "BattleCryExertsCount",
	}, nil)
}

// TestPactCalcSectionOutputs: CalcSections.lua:431-464: the Pacts section shows once a pact applies.
func TestPactCalcSectionOutputs(t *testing.T) {
	env := (&Calculator{PoB: loadCorpusBuild(t, "0207-occultist-winter-orb.xml")}).BuildOutput(OutputModeMain)
	out := env.Player.Output
	if out["LyciaUpTimeRatio"] == 0 || out["CreatePactOffensiveCalcSection"] != 1 {
		t.Fatalf("LyciaUpTimeRatio %v, CreatePactOffensiveCalcSection %v", out["LyciaUpTimeRatio"], out["CreatePactOffensiveCalcSection"])
	}
	var lines []string
	if e := env.Player.Breakdown.Data["LyciaUpTimeRatio"]; e != nil {
		lines = e.Lines
	}
	if len(lines) != 6 || !strings.HasSuffix(lines[0], "^8(number of Empowered)") || lines[5] != fmt.Sprintf("= %.2f%%", out["LyciaUpTimeRatio"]) {
		t.Errorf("LyciaUpTimeRatio breakdown %q", lines)
	}
}
