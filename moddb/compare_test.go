package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

func TestCompareModParams(t *testing.T) {
	base := func() mod.Mod {
		return mod.NewFloat("Damage", mod.TypeBase, 10).Tag(mod.Multiplier("Foo"))
	}

	testza.AssertTrue(t, CompareModParams(base(), base()))

	testza.AssertTrue(t, CompareModParams(base(), mod.NewFloat("Damage", mod.TypeBase, 999).Tag(mod.Multiplier("Foo"))))

	testza.AssertFalse(t, CompareModParams(base(), mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.Multiplier("Foo"))))

	testza.AssertFalse(t, CompareModParams(base(), mod.NewFloat("Damage", mod.TypeBase, 10).Tag(mod.Multiplier("Bar"))))
}
