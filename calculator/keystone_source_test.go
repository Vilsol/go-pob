package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestKeystoneKeepsGrantingItemAsSource pins ModTools.lua:226-237.
//
//	local fromTree = modObj.mod.source and not modObj.mod.source:lower():match("tree")
//	for _, mod in ipairs(env.spec.tree.keystoneMap[modObj.value].modList) do
//		modDB:AddMod(fromTree and modLib.setSource(mod, modObj.mod.source) or mod)
//	end
func TestKeystoneKeepsGrantingItemAsSource(t *testing.T) {
	requireGameData(t, "3.29")

	newEnv := func() *Environment {
		return &Environment{
			ModDB:          moddb.NewModDB(),
			KeystonesAdded: make(map[string]interface{}),
			Spec:           &PassiveSpec{TreeVersion: data.LatestTreeVersion},
		}
	}

	sourcesFor := func(env *Environment) []string {
		var out []string
		for name, mods := range env.ModDB.Mods {
			if name == "Keystone" {
				continue
			}
			for _, m := range mods {
				out = append(out, string(m.GetSource()))
			}
		}
		return out
	}

	t.Run("granted by an item, the stat lines carry the item's source", func(t *testing.T) {
		env := newEnv()
		env.ModDB.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Item:3:Kaom's Roots"))
		mergeKeystones(env, env.ModDB)

		got := sourcesFor(env)
		if len(got) == 0 {
			t.Fatal("REFUSING: the keystone merged no stat lines at all; this test cannot say " +
				"anything about their source")
		}
		for _, s := range got {
			if s != "Item:3:Kaom's Roots" {
				t.Errorf("stat line source = %q, want the granting item; ModTools.lua:231-233 "+
					"re-sources a keystone that did NOT come from the tree", s)
			}
		}
	})

	t.Run("granted by the tree, the source is left alone", func(t *testing.T) {
		env := newEnv()
		env.ModDB.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Tree:10661"))
		mergeKeystones(env, env.ModDB)

		for _, s := range sourcesFor(env) {
			if strings.HasPrefix(s, "Item:") {
				t.Errorf("a TREE-granted keystone was re-sourced to %q; the guard is "+
					"`not source:lower():match(\"tree\")`", s)
			}
		}
	})
}
