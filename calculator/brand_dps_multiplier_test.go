package calculator

import (
	"math"
	"os"
	"testing"
)

// TestBrandDPSMultiplier pins CalcOffence.lua:2423-2426.
//
//	if activeSkill.skillTypes[SkillType.Brand] and not skillData.countsAttachedBrandsInDamage then
//	end
func TestBrandDPSMultiplier(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	t.Run("applied", func(t *testing.T) {
		for _, build := range []string{
			"0078-elementalist-penance-brand.xml",
			"0090-elementalist-penance-brand-3.xml",
			"0398-hierophant-wintertide-brand.xml",
		} {
			t.Run(build, func(t *testing.T) {
				rec, ok := oracle[build]
				if !ok {
					t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
				}
				want, ok := rec.Output["SkillDPSMultiplier"]
				if !ok || want == 0 {
					t.Fatalf("REFUSING: PoB reports no nonzero SkillDPSMultiplier for %s", build)
				}

				env := corpusEnvForLoadedData(t, build)
				got, present := env.Player.Output["SkillDPSMultiplier"]
				if !present {
					t.Errorf("%s: we never write SkillDPSMultiplier at all; PoB reports %v", build, want)
					return
				}
				if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
					t.Errorf("%s: SkillDPSMultiplier = %v, PoB reports %v", build, got, want)
				}
				if c := env.Player.Output["AttachedBrandCount"]; math.Abs(got-c) > 1e-9 {
					t.Errorf("%s: SkillDPSMultiplier = %v but AttachedBrandCount = %v", build, got, c)
				}
			})
		}
	})

	t.Run("suppressed-on-wintertide-part-1", func(t *testing.T) {
		for _, build := range []string{
			"0063-occultist-wintertide-brand.xml",
			"0082-elementalist-wintertide-brand.xml",
			"0088-elementalist-wintertide-brand-3.xml",
		} {
			t.Run(build, func(t *testing.T) {
				rec, ok := oracle[build]
				if !ok {
					t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
				}
				if v, ok := rec.Output["SkillDPSMultiplier"]; ok && v != 0 {
					t.Fatalf("REFUSING: PoB reports SkillDPSMultiplier = %v for %s, so this build "+
						"no longer exercises the suppression and the fixture must be replaced", v, build)
				}
				if c, ok := rec.Output["AttachedBrandCount"]; !ok || c == 0 {
					t.Fatalf("REFUSING: %s has no nonzero AttachedBrandCount, so there is nothing "+
						"for the suppression to suppress", build)
				}

				env := corpusEnvForLoadedData(t, build)
				if got, present := env.Player.Output["SkillDPSMultiplier"]; present && got != 0 {
					t.Errorf("%s: we write SkillDPSMultiplier = %v where PoB writes none — "+
						"countsAttachedBrandsInDamage is not suppressing", build, got)
				}
			})
		}
	})
}
