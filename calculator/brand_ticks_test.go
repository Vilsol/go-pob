package calculator

import (
	"os"
	"testing"
)

// TestBrandTicksRespectsUnlimitedDuration pins CalcOffence.lua:2407.
//
//	if activeSkill.skillTypes[SkillType.Brand] and not skillModList:Flag(nil, "UnlimitedBrandDuration") then
//		output.BrandTicks = m_floor(output.Duration * output.HitSpeed)
//	end
func TestBrandTicksRespectsUnlimitedDuration(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	t.Run("unlimited-duration-writes-no-key", func(t *testing.T) {
		for _, build := range []string{
			"0078-elementalist-penance-brand.xml",
			"0090-elementalist-penance-brand-3.xml",
			"0100-elementalist-penance-brand-6.xml",
		} {
			t.Run(build, func(t *testing.T) {
				rec, ok := oracle[build]
				if !ok {
					t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
				}
				if v, ok := rec.Output["BrandTicks"]; ok && v != 0 {
					t.Fatalf("REFUSING: PoB reports BrandTicks = %v for %s, so this build no "+
						"longer exercises the unlimited-duration branch", v, build)
				}
				if c, ok := rec.Output["AttachedBrandCount"]; !ok || c == 0 {
					t.Fatalf("REFUSING: %s is not a brand build (no AttachedBrandCount), so "+
						"writing no BrandTicks proves nothing", build)
				}

				env := corpusEnvForLoadedData(t, build)
				if _, present := env.Player.Output["BrandTicks"]; present {
					t.Errorf("%s: we write BrandTicks where PoB writes no key — "+
						"UnlimitedBrandDuration is not being read", build)
				}
			})
		}
	})

	t.Run("limited-duration-still-writes-it", func(t *testing.T) {
		for _, build := range []string{
			"0408-inquisitor-penance-brand-of-dissipation.xml",
			"0506-assassin-penance-brand-of-dissipation.xml",
		} {
			t.Run(build, func(t *testing.T) {
				rec, ok := oracle[build]
				if !ok {
					t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
				}
				want, ok := rec.Output["BrandTicks"]
				if !ok || want == 0 {
					t.Fatalf("REFUSING: PoB reports no nonzero BrandTicks for %s, so this build "+
						"cannot control against over-suppression", build)
				}

				env := corpusEnvForLoadedData(t, build)
				got, present := env.Player.Output["BrandTicks"]
				if !present {
					t.Errorf("%s: we stopped writing BrandTicks entirely; PoB reports %v", build, want)
					return
				}
				if got != want {
					t.Errorf("%s: BrandTicks = %v, PoB reports %v", build, got, want)
				}
			})
		}
	})
}
