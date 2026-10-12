package calculator

import (
	"math"
	"os"
	"testing"
)

// TestMetreOneLinersArePublished pins the four remaining `<X>Metre` publications.
func TestMetreOneLinersArePublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, tc := range []struct {
		key    string
		radius string
		builds []string
	}{
		{
			key:    "TrapTriggerRadiusMetre",
			radius: "TrapTriggerRadius",
			builds: []string{
				"0014-trickster-spark-of-the-nova.xml",
				"0022-saboteur-blade-trap-of-greatswords.xml",
				"0026-ascendant-fire-trap-of-blasting.xml",
			},
		},
		{
			key:    "MineDetonationRadiusMetre",
			radius: "MineDetonationRadius",
			builds: []string{
				"0024-trickster-frostbite.xml",
				"0057-trickster-spark-of-the-nova-2.xml",
				"0114-occultist-icicle-mine-of-fanning.xml",
			},
		},
		{
			key:    "MineAuraRadiusMetre",
			radius: "MineAuraRadius",
			builds: []string{
				"0024-trickster-frostbite.xml",
				"0057-trickster-spark-of-the-nova-2.xml",
				"0114-occultist-icicle-mine-of-fanning.xml",
			},
		},
		{
			key:    "BrandAttachmentRangeMetre",
			radius: "BrandAttachmentRange",
			builds: []string{
				"0063-occultist-wintertide-brand.xml",
				"0078-elementalist-penance-brand.xml",
				"0398-hierophant-wintertide-brand.xml",
			},
		},
	} {
		t.Run(tc.key, func(t *testing.T) {
			for _, build := range tc.builds {
				t.Run(build, func(t *testing.T) {
					rec, ok := oracle[build]
					if !ok {
						t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
					}
					want, ok := rec.Output[tc.key]
					if !ok || want == 0 {
						t.Fatalf("REFUSING: PoB reports no nonzero %s for %s; an unwritten key "+
							"agrees with 0, so this fixture cannot fail", tc.key, build)
					}
					wantRadius, ok := rec.Output[tc.radius]
					if !ok || wantRadius == 0 {
						t.Fatalf("REFUSING: PoB reports no nonzero %s for %s", tc.radius, build)
					}

					env := corpusEnvForLoadedData(t, build)

					gotRadius := env.Player.Output[tc.radius]
					if math.Abs(gotRadius-wantRadius) > math.Abs(wantRadius)*oracleDumpTolerance {
						t.Fatalf("REFUSING: %s is a contaminated fixture — our %s = %v against "+
							"PoB's %v, so the quotient cannot be graded here. Pick a build whose "+
							"input already agrees.", build, tc.radius, gotRadius, wantRadius)
					}

					got, present := env.Player.Output[tc.key]
					if !present {
						t.Errorf("%s: we never write %s at all; PoB reports %v", build, tc.key, want)
						return
					}
					if math.IsNaN(got) || math.IsInf(got, 0) {
						t.Errorf("%s: %s = %v", build, tc.key, got)
						return
					}
					if math.Abs(got-gotRadius/10) > 1e-9 {
						t.Errorf("%s: %s = %v but %s/10 = %v", build, tc.key, got, tc.radius, gotRadius/10)
					}
					if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
						t.Errorf("%s: %s = %v, PoB reports %v", build, tc.key, got, want)
					}
				})
			}
		})
	}
}
