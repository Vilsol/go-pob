package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

//	local unscalable = false
//	for _, effects in ipairs(mod) do
//		if effects.unscalable then unscalable = true break end
//	end
//	if scale == 1 or unscalable then self:AddMod(mod) else ... end

func unscalableMod(value float64) mod.Mod {
	return mod.NewFloat("AvoidCurse", mod.TypeBase, value).
		Tag(mod.GlobalEffect("Aura").Unscalable(true))
}

func scalableMod(value float64) mod.Mod {
	return mod.NewFloat("AvoidCurse", mod.TypeBase, value).
		Tag(mod.GlobalEffect("Aura"))
}

func TestModListScaleAddListSkipsUnscalable(t *testing.T) {
	src := NewModList()
	src.AddMod(unscalableMod(100))

	dst := NewModList()
	dst.ScaleAddList(src, 2)

	testza.AssertEqual(t, 100.0, dst.Sum(mod.TypeBase, nil, "AvoidCurse"))
}

func TestModDBScaleAddListSkipsUnscalable(t *testing.T) {
	src := NewModList()
	src.AddMod(unscalableMod(100))

	db := NewModDB()
	db.ScaleAddList(src, 2)

	testza.AssertEqual(t, 100.0, db.Sum(mod.TypeBase, nil, "AvoidCurse"))
}

// TestScaleAddListStillScalesWithoutUnscalable: The CONTROL.
func TestScaleAddListStillScalesWithoutUnscalable(t *testing.T) {
	src := NewModList()
	src.AddMod(scalableMod(100))

	dst := NewModList()
	dst.ScaleAddList(src, 2)
	testza.AssertEqual(t, 200.0, dst.Sum(mod.TypeBase, nil, "AvoidCurse"))

	db := NewModDB()
	db.ScaleAddList(src, 2)
	testza.AssertEqual(t, 200.0, db.Sum(mod.TypeBase, nil, "AvoidCurse"))
}

// TestUnscalableFoundOnNonFirstTag: ModStore.lua:47 iterates ALL tags.
func TestUnscalableFoundOnNonFirstTag(t *testing.T) {
	src := NewModList()
	src.AddMod(mod.NewFloat("AvoidCurse", mod.TypeBase, 100).
		Tag(mod.Condition("Active")).
		Tag(mod.GlobalEffect("Aura").Unscalable(true)))
	src.Conditions = map[string]bool{"Active": true}

	dst := NewModList()
	dst.Conditions = map[string]bool{"Active": true}
	dst.ScaleAddList(src, 2)

	testza.AssertEqual(t, 100.0, dst.Sum(mod.TypeBase, nil, "AvoidCurse"))
}
