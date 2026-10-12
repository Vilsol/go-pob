package calculator

import (
	"strings"
	"testing"
)

// TestPactDamageReachesTheSkill pins CalcOffence.lua:2845-2849, the DAMAGE half of the Pact block.
func TestPactDamageReachesTheSkill(t *testing.T) {
	damageStats := []string{"TotalMin", "TotalMax", "AverageDamage"}

	t.Run("Lycia", func(t *testing.T) {
		assertBuildStats(t, "0371-occultist-winter-orb-5.xml", damageStats, nil)
	})

	t.Run("Beidat", func(t *testing.T) {
		assertBuildStats(t, "0890-hierophant-firestorm-of-meteors.xml", damageStats, nil)
	})

	t.Run("K'Tash", func(t *testing.T) {
		assertBuildStats(t, "0825-elementalist-vaal-flameblast.xml", damageStats, nil)
	})

	t.Run("a Winter Orb build with no Pact is untouched", func(t *testing.T) {
		assertBuildStats(t, "0075-elementalist-winter-orb.xml", damageStats, func(t *testing.T, xml string) {
			t.Helper()
			if strings.Contains(xml, "Pact of ") {
				t.Fatalf("REFUSING: the control build now carries a Pact, so it no longer " +
					"controls for anything; pick another")
			}
		})
	})
}
