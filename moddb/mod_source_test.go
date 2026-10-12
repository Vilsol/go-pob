package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func srcCfg(source string) *ListCfg {
	return &ListCfg{Source: utils.Ptr(mod.Source(source))}
}

func TestModDBSourcePrefixMatch(t *testing.T) {
	m := NewModDB()
	m.AddMod(mod.NewFloat("X", mod.TypeBase, 100).Source("Tree:12345"))

	testza.AssertEqual(t, float64(100), m.Sum(mod.TypeBase, srcCfg("Tree"), "X"))

	testza.AssertEqual(t, float64(0), m.Sum(mod.TypeBase, srcCfg("Item"), "X"))

	m2 := NewModDB()
	m2.AddMod(mod.NewFloat("X", mod.TypeBase, 100).Source("Base"))
	testza.AssertEqual(t, float64(100), m2.Sum(mod.TypeBase, srcCfg("Base"), "X"))
}

func TestModListSourcePrefixMatch(t *testing.T) {
	m := NewModList()
	m.AddMod(mod.NewFloat("X", mod.TypeBase, 100).Source("Tree:12345"))

	testza.AssertEqual(t, float64(100), m.Sum(mod.TypeBase, srcCfg("Tree"), "X"))
	testza.AssertEqual(t, float64(0), m.Sum(mod.TypeBase, srcCfg("Item"), "X"))

	m2 := NewModList()
	m2.AddMod(mod.NewFloat("X", mod.TypeBase, 100).Source("Base"))
	testza.AssertEqual(t, float64(100), m2.Sum(mod.TypeBase, srcCfg("Base"), "X"))
}

func TestModDBAddDBAppends(t *testing.T) {
	a := NewModDB()
	a.AddMod(mod.NewFloat("X", mod.TypeBase, 100))

	b := NewModDB()
	b.AddMod(mod.NewFloat("X", mod.TypeBase, 50))

	a.AddDB(b)

	testza.AssertEqual(t, float64(150), a.Sum(mod.TypeBase, nil, "X"))
}
