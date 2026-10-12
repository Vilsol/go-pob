package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestProjectilesChainAnAdditionalTimeRow pins ModParser.lua:4506.
func TestProjectilesChainAnAdditionalTimeRow(t *testing.T) {
	e := ParseMod("Projectiles Chain an additional time", false)
	if e == nil || len(e.ModList) != 1 || e.Extra != "" {
		t.Fatalf("got %v", e)
	}
	m := e.ModList[0]
	if m.Name() != "ChainCountMax" || m.Type() != mod.TypeBase {
		t.Fatalf("got %s %s, want ChainCountMax BASE", m.Name(), m.Type())
	}
	if v := m.Value().Float(); v != 1 {
		t.Errorf("value = %v, want 1", v)
	}
	if m.Flags()&mod.MFlagProjectile == 0 {
		t.Errorf("flags = %d, want MFlagProjectile set", m.Flags())
	}
}
