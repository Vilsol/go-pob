package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestHostileMinionTakesEnemyModifiers pins CalcPerform.lua:1980-1999 on an edited build PoB dumped: with Penance
// Mark as the main skill the hostile phantasm takes the player's EnemyModifier mods and none of its MinionModifier.
func TestHostileMinionTakesEnemyModifiers(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.30")
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), "0161-assassin-viper-strike-of-the-mamba.xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	s := string(raw)
	const line = "{crafted}+28 to maximum Energy Shield\n"
	for _, old := range []string{`mainSocketGroup="3"`, line} {
		if n := strings.Count(s, old); n != 1 {
			t.Fatalf("REFUSING: %q occurs %d times", old, n)
		}
	}
	s = strings.Replace(s, `mainSocketGroup="3"`, `mainSocketGroup="6"`, 1)
	s = strings.Replace(s, line, line+"Minions have 40% increased maximum Life\nNearby Enemies have -10% to all Resistances\n", 1)
	b, err := builds.ParseBuild([]byte(s))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	env := (&Calculator{PoB: b}).BuildOutput(OutputModeMain)
	if env.Minion == nil || env.Minion.MinionData == nil || !env.Minion.MinionData.Hostile {
		t.Fatalf("PREMISE BROKEN: the main skill's minion is not hostile")
	}
	// PoB's dump of this edit.
	for stat, want := range map[string]float64{"Life": 45883, "FireResist": 66, "ChaosResist": 10} {
		if got := env.Minion.Output[stat]; got != want {
			t.Errorf("minion %s = %v, PoB %v", stat, got, want)
		}
	}
}
