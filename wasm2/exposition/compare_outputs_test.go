package exposition

import (
	"testing"

	"github.com/Vilsol/go-pob/calculator2"
)

// The stat list's condFuncs read output's booleans (e.g. a cost per second shown only when a flag is set); item, node
// and compare tooltips grade the same outputs, so a true flag must reach them as 1 and a false one stay absent.
func TestCompareOutputsCarryTrueFlags(t *testing.T) {
	output := map[string]float64{"Life": 100}
	env := &calculator2.Environment{Player: &calculator2.Actor{
		Output:      output,
		OutputFlags: map[string]bool{"ShowBlockEffect": true, "Off": false},
	}}
	got := compareOutputs(env).Output
	if got["Life"] != 100 || got["ShowBlockEffect"] != 1 {
		t.Fatalf("compareOutputs = %v, want Life 100 and ShowBlockEffect 1", got)
	}
	if _, ok := got["Off"]; ok {
		t.Fatalf("a false flag reached the outputs: %v", got)
	}
	if _, ok := output["ShowBlockEffect"]; ok {
		t.Fatal("compareOutputs wrote into the actor's own output table")
	}
}
