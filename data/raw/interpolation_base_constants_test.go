package raw

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

// TestInterpolationBasesReproduceBaseResolvedValues is the standing decider for the two EffectivenessCostConstants values this tree RECONSTRUCTS.
func TestInterpolationBasesReproduceBaseResolvedValues(t *testing.T) {
	if err := poe.InitializeAll(DataContext(t.Context()), LatestVersion, AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	setByKey := make(map[int]*poe.GrantedEffectStatSet, len(poe.GrantedEffectStatSets))
	for _, set := range poe.GrantedEffectStatSets {
		setByKey[set.Key] = set
	}

	seen := map[int]int{}
	exact := map[int]int{}
	for _, row := range poe.GrantedEffectStatSetsPerLevels {
		set := setByKey[row.StatSet]
		if set == nil {
			continue
		}

		baseEffectiveness := set.BaseEffectiveness
		if baseEffectiveness == 0 {
			baseEffectiveness = 1
		}
		actorLevel := math.Max(float64(row.PlayerLevelReq), 1)
		available := (3.885209 + 0.360246*(actorLevel-1)) * baseEffectiveness *
			math.Pow(1+set.IncrementalEffectiveness, actorLevel-1)

		for i := range row.FloatStats {
			if i >= len(row.InterpolationBases) || i >= len(row.FloatStatsValues) || i >= len(row.BaseResolvedValues) {
				continue
			}
			key := row.InterpolationBases[i]
			seen[key]++
			value := row.FloatStatsValues[i] / math.Max(InterpolationBaseValue(key), 0.00001)
			if math.Round(available*value) == float64(row.BaseResolvedValues[i]) {
				exact[key]++
			}
		}
	}

	for key := range interpolationBaseValues {
		if seen[key] == 0 {
			t.Fatalf("REFUSING: EffectivenessCostConstants key %d is used by no stat row; the "+
				"population this constant was recovered from has emptied", key)
		}
		rate := float64(exact[key]) / float64(seen[key])
		if rate < 0.95 {
			t.Errorf("EffectivenessCostConstants key %d (value %v): only %d of %d entries (%.2f%%) "+
				"reproduce BaseResolvedValues; the reconstructed constant is wrong",
				key, interpolationBaseValues[key], exact[key], seen[key], 100*rate)
		}
	}
}
