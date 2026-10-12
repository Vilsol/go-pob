package calculator

import (
	"math"
	"os"
	"testing"
)

// TestRageStacksFollowPoB pins CalcPerform.lua:882-897 — the stack arithmetic the rage block runs once its gate is open, and the flag it raises at the cap.
func TestRageStacksFollowPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	t.Run("HaveMaximumRage is raised at the cap", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0007-elementalist-storm-burst-of-repulsion.xml")
		if got, want := env.Player.Output["Rage"], env.Player.Output["MaximumRage"]; got != want {
			t.Fatalf("fixture no longer sits at the cap: Rage %v, MaximumRage %v", got, want)
		}
		if !env.Player.ModDB.Flag(nil, "Condition:HaveMaximumRage") {
			t.Errorf("Condition:HaveMaximumRage is FALSE while Rage == MaximumRage == %v; "+
				"CalcPerform.lua:896-898 raises it whenever stacks == maxStacks",
				env.Player.Output["Rage"])
		}
	})

	t.Run("CONTROL: not raised when the gate is shut", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0004-trickster-lightning-tendrils-of-eccentricity.xml")
		if env.Player.ModDB.Flag(nil, "Condition:HaveMaximumRage") {
			t.Errorf("Condition:HaveMaximumRage is raised on a build with no rage source — " +
				"0 == 0 satisfies `stacks == maxStacks`, so the flag must be inside the gated block")
		}
	})

	t.Run("maxStacks keeps its value where no MORE applies", func(t *testing.T) {
		env := corpusEnvForLoadedData(t, "0007-elementalist-storm-burst-of-repulsion.xml")
		if got := env.Player.Output["MaximumRage"]; math.Abs(got-30) > 0.001 {
			t.Errorf("MaximumRage = %v, want 30", got)
		}
	})
}
