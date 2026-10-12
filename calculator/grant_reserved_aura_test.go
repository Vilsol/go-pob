package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestGrantReservedPoolAsAura covers the "Grants Armour equal to N% of your reserved Life" path in doActorLifeManaReservation.
func TestGrantReservedPoolAsAura(t *testing.T) {
	actor := &Actor{
		ModDB:  moddb.NewModDB(),
		Output: map[string]float64{},
	}
	actor.Output["Life"] = 1000
	actor.Output["reserved_LifeBase"] = 500
	actor.Output["reserved_LifePercent"] = 0

	actor.ModDB.AddMod(mod.NewList("GrantReservedLifeAsAura", mod.GrantReservedPoolAsAura{
		Mod: mod.NewFloat("Armour", mod.TypeBase, 100.0/100),
	}))

	doActorLifeManaReservation(actor, true)

	extraAuras := actor.ModDB.List(nil, "ExtraAura")
	if len(extraAuras) != 1 {
		t.Fatalf("got %d ExtraAura mods, want 1", len(extraAuras))
	}

	aura, ok := extraAuras[0].(mod.ExtraAura)
	if !ok {
		t.Fatalf("ExtraAura list value is %T, want mod.ExtraAura", extraAuras[0])
	}
	if got := aura.Mod.Value().Float(); got != 500 {
		t.Fatalf("granted Armour = %v, want 500", got)
	}
	src := actor.ModDB.List(nil, "GrantReservedLifeAsAura")[0].(mod.GrantReservedPoolAsAura)
	if got := src.Mod.Value().Float(); got != 1 {
		t.Fatalf("source mod value = %v, want 1 (it was scaled in place)", got)
	}
}
