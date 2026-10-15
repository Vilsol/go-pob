package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestTwoTypeDamageFromHitsTakenAsRow pins ModParser.lua:5316-5324 (upstream #10230, Rhex).
func TestTwoTypeDamageFromHitsTakenAsRow(t *testing.T) {
	entry := ParseMod("100% of Cold and Lightning Damage from Hits taken as Fire Damage", false)
	if strings.TrimSpace(entry.Extra) != "" {
		t.Fatalf("unparsed remainder %q", entry.Extra)
	}
	got := map[string]float64{}
	for _, m := range entry.ModList {
		if m.Type() == mod.TypeBase {
			got[m.Name()] = m.Value().Float()
		}
	}
	if len(entry.ModList) != 2 || got["ColdDamageFromHitsTakenAsFire"] != 100 || got["LightningDamageFromHitsTakenAsFire"] != 100 {
		t.Errorf("got %v, want Cold and Lightning DamageFromHitsTakenAsFire BASE 100", got)
	}
}
