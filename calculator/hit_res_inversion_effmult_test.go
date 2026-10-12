package calculator

import (
	"os"
	"testing"
)

// TestHitResInversionEffMult pins the consumer half of CalcOffence.lua:3478-3505 — the `invertChance` blend of the enemy's resistance inside the hit `effMult`.
func TestHitResInversionEffMult(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	for _, build := range []string{
		"0292-hierophant-kinetic-fusillade-2",
		"0406-reliquarian-static-strike",
	} {
		t.Run(build, func(t *testing.T) {
			assertMainHandStats(t, oracle, build,
				"FireHitAverage", "ColdHitAverage", "LightningHitAverage")
		})
	}
}
