package calculator

import "testing"

// TestMinionLimitHasOneWriter pins the Active*Limit family to PoB's single write site.
func TestMinionLimitHasOneWriter(t *testing.T) {
	for _, name := range []string{
		"0397-hierophant-bladefall-of-impaling.xml",
		"0938-luminary-perforate-of-duality.xml",
		"1054-chieftain-shield-crush-of-the-chieftain-2.xml",
		"1159-reliquarian-power-siphon.xml",
		"1222-chieftain-shield-crush-of-the-chieftain-3.xml",
		"1457-juggernaut-herald-of-agony.xml",
		"1482-chieftain-generals-cry.xml",
	} {
		t.Run(name, func(t *testing.T) {
			assertBuildStats(t, name, []string{"ActiveSpectreLimit"}, nil)
		})
	}
}
