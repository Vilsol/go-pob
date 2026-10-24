package main

import "testing"

// TestEmitTagPercentStatCarriesActorAndFloor pins BOTH PercentStat arms — ssmRun.emitTag and sbmEmitTag — against the fields ModStore.lua actually evaluates.
func TestEmitTagPercentStatCarriesActorAndFloor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tag  map[string]any
		want string
	}{
		{"actor", map[string]any{
			"type": "PercentStat", "stat": "Life", "percent": mustNumber("1"), "actor": "parent",
		}, `mod.PercentStat("Life", 1).Actor("parent")`},
		{"percentVar + floor", map[string]any{
			"type": "PercentStat", "stat": "Life", "percentVar": "KiloavaRecoverOnBlock", "floor": true,
		}, `mod.PercentStatVar("Life", "KiloavaRecoverOnBlock").Floor()`},
		{"neighbour: plain percent", map[string]any{
			"type": "PercentStat", "stat": "ManaUnreserved", "percent": mustNumber("1"),
		}, `mod.PercentStat("ManaUnreserved", 1)`},
		{"neighbour: plain percentVar", map[string]any{
			"type": "PercentStat", "stat": "ManaUnreserved", "percentVar": "ArcaneCloakConsumedMana",
		}, `mod.PercentStatVar("ManaUnreserved", "ArcaneCloakConsumedMana")`},
	} {
		t.Run("skill-stat-map/"+tc.name, func(t *testing.T) {
			got, err := (&ssmRun{}).emitTag(tc.tag)
			if err != nil {
				t.Fatalf("emitTag(%v) errored: %v", tc.tag, err)
			}
			if got != tc.want {
				t.Fatalf("emitTag(%v)\n  got  %s\n  want %s", tc.tag, got, tc.want)
			}
		})
		t.Run("skill-base-mods/"+tc.name, func(t *testing.T) {
			got, err := sbmEmitTag(tc.tag, nil, nil)
			if err != nil {
				t.Fatalf("sbmEmitTag(%v) errored: %v", tc.tag, err)
			}
			if got != tc.want {
				t.Fatalf("sbmEmitTag(%v)\n  got  %s\n  want %s", tc.tag, got, tc.want)
			}
		})
	}
}
