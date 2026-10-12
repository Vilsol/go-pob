package raw

import (
	"math"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestLevelStatRowPairingMatchesPoB pins the level row <-> stat-set row pairing against PoB's own exported skill data.
func TestLevelStatRowPairingMatchesPoB(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	vaal := map[int][2]*float64{
		1:  {ptr(0.75), ptr(0.75)},
		2:  {ptr(0.804), ptr(0.804)},
		19: {ptr(1.716), ptr(1.716)},
		20: {ptr(1.77), ptr(1.77)},
		21: {ptr(1.824), ptr(1.824)},
	}

	strike := map[int][2]*float64{
		1:  {ptr(1.985), ptr(1.985)},
		19: {ptr(4.373), ptr(4.373)},
		39: {ptr(8.693), ptr(8.693)},
		40: {ptr(8.878), ptr(8.878)},
	}

	for _, tc := range []struct {
		id   string
		want map[int][2]*float64
	}{
		{"VaalLightningStrike", vaal},
		{"LightningStrike", strike},
	} {
		ge := poe.GrantedEffectByID(tc.id)
		if ge == nil {
			t.Fatalf("granted effect %s not found", tc.id)
		}

		levels := GetCalculatedGrantedEffect(ge).GetCalculatedLevels()
		for level, want := range tc.want {
			got := levels[level]
			if got == nil {
				t.Fatalf("%s: level %d missing", tc.id, level)
			}
			assertOptionalFloat(t, tc.id, level, "baseMultiplier", want[0], got.BaseMultiplier)
			assertOptionalFloat(t, tc.id, level, "damageEffectiveness", want[1], got.DamageEffectiveness)
		}
	}

	vls := GetCalculatedGrantedEffect(poe.GrantedEffectByID("VaalLightningStrike"))
	testza.AssertEqual(t, []float64{9}, vls.GetCalculatedLevels()[19].Values)
}

func ptr(v float64) *float64 {
	return &v
}

func assertOptionalFloat(t *testing.T, id string, level int, field string, want, got *float64) {
	t.Helper()

	switch {
	case want == nil && got != nil:
		t.Errorf("%s level %d: %s = %v, PoB exports none", id, level, field, *got)
	case want != nil && got == nil:
		t.Errorf("%s level %d: %s missing, PoB exports %v", id, level, field, *want)
	case want != nil && got != nil && math.Abs(*want-*got) > 1e-9:
		t.Errorf("%s level %d: %s = %v, PoB exports %v", id, level, field, *got, *want)
	}
}
