package calculator

import "testing"

// TestBloodSacramentReservationPerStage pins CalcPerform.lua:1994-1998: Blood Sacrament multiplies its reservation by the channelled stage count.
func TestBloodSacramentReservationPerStage(t *testing.T) {
	for _, name := range []string{"9116-ladder-BiboooTheRock_4194.BibooTheGoat.xml", "4879-ladder-Harwinnam_4002.AFNiwrah.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"LifeReserved", "LifeUnreserved"}, nil)
		})
	}
}
