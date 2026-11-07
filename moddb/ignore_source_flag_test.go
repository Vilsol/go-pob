package moddb

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// ModDB.lua:302-303 (PoB-PoE2): with ignoreSourceInCheckConditions a ModDB's Flag matches a mod of any source, so a
// condition a "Tree" source total checks still sees a flag the config set.
func TestFlagIgnoresSourceInCheckConditions(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFlag("Condition:WeaponSet1", true).Source("Base"))
	db.AddMod(mod.NewFloat("Life", mod.TypeBase, 3).Source("Tree:1").Tag(mod.Condition("WeaponSet1")))
	tree := mod.Source("Tree")
	if db.Flag(&ListCfg{Source: &tree}, "Condition:WeaponSet1") {
		t.Error("a Tree-source Flag matched a Base mod")
	}
	if got := db.Sum(mod.TypeBase, &ListCfg{Source: &tree}, "Life"); got != 0 {
		t.Errorf("Tree total %v, want 0 without the cfg", got)
	}
	cfg := &ListCfg{Source: &tree, IgnoreSourceInCheckConditions: true}
	if !db.Flag(cfg, "Condition:WeaponSet1") {
		t.Error("ignoreSourceInCheckConditions: Flag still filtered by source")
	}
	if got := db.Sum(mod.TypeBase, cfg, "Life"); got != 3 {
		t.Errorf("ignoreSourceInCheckConditions: Tree total %v, want 3", got)
	}
}
