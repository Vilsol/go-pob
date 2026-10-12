package calculator

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	sacredWispsProducingFloor  = 164
	mirageArcherProducingFloor = 119
)

// TestSacredWispsMirageProducesOutput pins CalcMirages.lua:282-343, the Sacred Wisps arm.
func TestSacredWispsMirageProducesOutput(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	corpusDir, names := corpusNamesForLoadedData(t)

	var wispSeen, wispProducing, archerSeen, archerProducing int
	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		if rec.Output["MirageDPS"] == 0 {
			continue
		}
		raw, err := os.ReadFile(corpusDir + "/" + name)
		if err != nil {
			continue
		}
		wisps := strings.Contains(string(raw), "SupportSacredWisps")

		env := corpusEnvForLoadedData(t, name)
		producing := env.Player.Output["MirageDPS"] != 0
		if wisps {
			wispSeen++
			if producing {
				wispProducing++
			}
			continue
		}
		archerSeen++
		if producing {
			archerProducing++
		}
	}

	if wispSeen == 0 || archerSeen == 0 {
		t.Fatalf("REFUSING: Sacred Wisps population %d, Mirage Archer control %d — a floor over "+
			"an empty set asserts nothing", wispSeen, archerSeen)
	}

	fmt.Printf("=== SACRED WISPS MIRAGE === wisps %d/%d producing · archer control %d/%d producing\n",
		wispProducing, wispSeen, archerProducing, archerSeen)

	if wispProducing < sacredWispsProducingFloor {
		t.Errorf("Sacred Wisps builds producing a MirageDPS: %d of %d, floor %d. "+
			"CalcMirages.lua:282-343's arm has stopped running, or "+
			"SkillStatMap.lua:288's summon_sacred_wisps_on_hit is emitting an empty mod list "+
			"again", wispProducing, wispSeen, sacredWispsProducingFloor)
	}
	if archerProducing < mirageArcherProducingFloor {
		t.Errorf("CONTROL MOVED: Mirage Archer builds producing a MirageDPS: %d of %d, floor %d. "+
			"This change does not touch that arm, so a drop here means it touched something else",
			archerProducing, archerSeen, mirageArcherProducingFloor)
	}
}
