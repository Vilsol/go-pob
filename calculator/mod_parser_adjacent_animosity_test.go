package calculator

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestAdjacentAnimosityRow pins ModParser.lua:4585.
func TestAdjacentAnimosityRow(t *testing.T) {
	entry := ParseMod("Projectiles deal 40% increased Damage with Hits to targets at the start of their movement, reducing to 0% as they travel farther", false)
	if strings.TrimSpace(entry.Extra) != "" || len(entry.ModList) != 1 {
		t.Fatalf("extra %q, %d mods", entry.Extra, len(entry.ModList))
	}
	m := entry.ModList[0]
	if m.Name() != "Damage" || m.Type() != mod.TypeIncrease || m.Value().Float() != 40 {
		t.Errorf("got %s %s %v", m.Name(), m.Type(), m.Value().Float())
	}
	if m.Flags() != mod.MFlagHit|mod.MFlagProjectile {
		t.Errorf("flags %v, want Hit|Projectile", m.Flags())
	}
	want := []mod.Tag{mod.DistanceRamp([][]float64{{35, 1}, {70, 0}})}
	if !reflect.DeepEqual(m.Tags(), want) {
		t.Errorf("tags %#v, want %#v", m.Tags(), want)
	}
}
