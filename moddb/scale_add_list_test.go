package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

func TestScaleAddList(t *testing.T) {
	src := NewModList()
	src.AddMod(mod.NewFloat("X", mod.TypeBase, 100))
	src.AddMod(mod.NewFlag("F", true))

	dest := NewModList()
	dest.ScaleAddList(src, 0.5)

	testza.AssertEqual(t, float64(50), dest.Sum(mod.TypeBase, nil, "X"))
	testza.AssertTrue(t, dest.Flag(nil, "F"))

	testza.AssertEqual(t, float64(100), src.Sum(mod.TypeBase, nil, "X"))
}

func TestNewMod(t *testing.T) {
	m := NewModList()
	m.NewMod("Speed", mod.TypeIncrease, 20, mod.SourceConfig)
	testza.AssertEqual(t, float64(20), m.Sum(mod.TypeIncrease, nil, "Speed"))
}
