package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestPayableCostRows pins ModParser.lua:3056-3059, :3120-3123 and :5640-5651.
func TestPayableCostRows(t *testing.T) {
	for _, tc := range []struct{ line, pool, cost string }{
		{"Spells you cast yourself gain added Physical Damage equal to 25% of Life Cost, if Life Cost is not higher than the maximum you could spend", "LifeUnreserved", "LifeCost"},
		{"Skills gain added Chaos Damage equal to 25% of Life Cost, if Life Cost is not higher than the maximum you could spend", "LifeUnreserved", "LifeCost"},
		{"Skills gain added Chaos Damage equal to 25% of Mana Cost, if Mana Cost is not higher than the maximum you could spend", "ManaCostPayablePool", "ManaCost"},
		{"Arc and Crackling Lance gains added Cold Damage equal to 25% of Mana Cost, if Mana Cost is not higher than the maximum you could spend", "ManaCostPayablePool", "ManaCost"},
		{"Forbidden Rite and Dark Pact gains added Chaos Damage equal to 25% of Mana Cost, if Mana Cost is not higher than the maximum you could spend", "ManaCostPayablePool", "ManaCost"},
	} {
		entry := ParseMod(tc.line, false)
		if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 2 {
			t.Errorf("%q: extra %q, %d mods", tc.line, entry.Extra, len(entry.ModList))
			continue
		}
		var floor, threshold bool
		for _, tag := range entry.ModList[0].Tags() {
			switch v := tag.(type) {
			case *mod.PercentStatTag:
				floor = v.TagFloor
			case *mod.StatThresholdTag:
				threshold = v.Stat == tc.pool && v.TagThresholdStat == tc.cost && v.ThresholdPercent == 0
			}
		}
		if !floor || !threshold {
			t.Errorf("%q: floor %v, threshold %s >= %s with no percent %v", tc.line, floor, tc.pool, tc.cost, threshold)
		}
	}
}

// TestManaCostPayablePool pins CalcOffence.lua:1897-1898; the value is the pin-3.29-2026-10 oracle's.
func TestManaCostPayablePool(t *testing.T) {
	assertPlayerOutputs(t, "0001-necromancer-unknown.xml", map[string]float64{"ManaCostPayablePool": 97})
}
