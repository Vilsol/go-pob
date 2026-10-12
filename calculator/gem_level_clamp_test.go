package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

// TestValidateGemLevelClampsToPoBsCeiling pins calcLib.validateGemLevel's first arm (CalcTools.lua:47-49).
//
//	if #grantedEffect.levels > 0 then
//		gemInstance.level = m_min(#grantedEffect.levels, gemInstance.level)
//	end
func TestValidateGemLevelClampsToPoBsCeiling(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	effect := func(id string) *GrantedEffect {
		ge := poe.GrantedEffectByID(id)
		if ge == nil {
			t.Fatalf("REFUSING: %q is not a granted effect in the loaded data — wrong fixture", id)
		}
		return &GrantedEffect{Raw: ge}
	}

	t.Run("dense", func(t *testing.T) {
		ge := effect("Arc")
		levels := raw.GetCalculatedGrantedEffect(ge.Raw).GetCalculatedLevels()
		if _, ok := levels[len(levels)]; !ok {
			t.Fatalf("REFUSING: Arc's level map is not dense 1..%d — wrong fixture", len(levels))
		}
		want := len(levels)

		gem := &GemEffect{GrantedEffect: ge, Level: want + 5}
		CalcValidateGemLevel(gem)
		if gem.Level != want {
			t.Errorf("level %d clamped to %d, PoB clamps to #levels = %d", want+5, gem.Level, want)
		}

		gem = &GemEffect{GrantedEffect: ge, Level: want}
		CalcValidateGemLevel(gem)
		if gem.Level != want {
			t.Errorf("level %d, already valid, was rewritten to %d", want, gem.Level)
		}
	})

	t.Run("sparse", func(t *testing.T) {
		ge := effect("IceNovaTriggered")
		levels := raw.GetCalculatedGrantedEffect(ge.Raw).GetCalculatedLevels()
		if _, ok := levels[1]; ok {
			t.Fatal("REFUSING: IceNovaTriggered now has a level 1, so it no longer exercises " +
				"the sequence-length case — wrong fixture")
		}
		if len(levels) != 1 {
			t.Fatalf("REFUSING: IceNovaTriggered has %d levels, want the single sparse one", len(levels))
		}

		gem := &GemEffect{GrantedEffect: ge, Level: 5}
		CalcValidateGemLevel(gem)
		if _, ok := levels[gem.Level]; !ok {
			t.Errorf("level 5 resolved to %d, which is not a level this effect has", gem.Level)
		}
	})
}
