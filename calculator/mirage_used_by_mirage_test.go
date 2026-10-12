package calculator

import "testing"

// TestMirageCopyIsUsedByMirage pins CalcMirages.lua:39: the mirage copy carries skillCond usedByMirage.
func TestMirageCopyIsUsedByMirage(t *testing.T) {
	for _, name := range []string{"3502-ladder-fjwilcox_0296.OceanVibes.xml", "9691-ladder-DoctorManhattanv2_3654.culler_zero_rarity.xml"} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"MirageDPS", "CullingDPS", "CombinedDPS"}, nil)
		})
	}
}
