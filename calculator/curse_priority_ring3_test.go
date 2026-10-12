package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestCursePriorityCollapsesRing3LikeRing2 pins the second disjunct of CalcPerform.lua:588.
//
//	if source ~= "" and (slotPriority == data.cursePriority["Ring 2"]
//	                     or slotPriority == data.cursePriority["Ring 3"]) then
func TestCursePriorityCollapsesRing3LikeRing2(t *testing.T) {
	priorityFor := func(slot, source string) float64 {
		return determineCursePriority("Despair", &ActiveSkill{
			SocketGroup: &pob.Skill{Source: source, Slot: slot},
		})
	}

	ring1 := priorityFor("Ring 1", "Tabula Rasa")
	for _, slot := range []string{"Ring 2", "Ring 3"} {
		if got := priorityFor(slot, "Tabula Rasa"); got != ring1 {
			t.Errorf("equipment curse in %q has priority %v, want %v (Ring 1) — "+
				"CalcPerform.lua:588 collapses Ring 2 AND Ring 3 onto Ring 1", slot, got, ring1)
		}
	}

	if priorityFor("Ring 3", "") == priorityFor("Ring 1", "") {
		t.Errorf("CONTROL: with no source, Ring 3 and Ring 1 must keep DIFFERENT priorities — " +
			"the collapse is gated on source, and that gate has been lost")
	}
}
