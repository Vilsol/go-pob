package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestDurationModifierIsClamped pins `durationMod = m_max(durationMod, 0)`.
func TestDurationModifierIsClamped(t *testing.T) {
	requireGameData(t, "3.29")

	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("Duration", mod.TypeIncrease, -150))

	got := CalcMod(db, nil, "Duration")
	if got >= 0 {
		t.Fatalf("REFUSING: calcLib.mod returned %v for -150%% increased duration; this fixture "+
			"cannot exercise a clamp that never sees a negative", got)
	}

	if clamped := durationModClamped(db, nil, "Duration"); clamped != 0 {
		t.Errorf("clamped duration modifier = %v, want 0 (CalcOffence.lua:302 and four siblings "+
			"apply m_max(durationMod, 0); v2.18.1 applied none)", clamped)
	}
}
