package calculator

import (
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestAbyssKeystonesResolveByName: PassiveTree.lua:708-716 puts every `ks` legion node into keystoneMap.
func TestAbyssKeystonesResolveByName(t *testing.T) {
	fromKeystones := make(map[string]bool, len(data.LegionKeystones))
	for _, node := range data.LegionKeystones {
		fromKeystones[node.DN] = true
	}
	added := make([]string, 0, 4)
	for name, node := range legionKeystonesByName() {
		if !fromKeystones[name] {
			added = append(added, node.ID)
		}
	}
	sort.Strings(added)
	want := []string{"abyss_ghastly_keystone", "abyss_hypnotic_keystone", "abyss_murderous_keystone", "abyss_searching_keystone"}
	if len(added) != len(want) {
		t.Fatalf("PassiveTree.lua:708-716 adds %v beyond data.LegionKeystones, want %v", added, want)
	}
	for i := range want {
		if added[i] != want[i] {
			t.Fatalf("PassiveTree.lua:708-716 adds %v beyond data.LegionKeystones, want %v", added, want)
		}
	}

	env := &Environment{Spec: &PassiveSpec{TreeVersion: data.LatestTreeVersion}}
	list := keystoneModList(env, data.LatestTreeVersion, "Overwhelming Hate")
	if list == nil {
		t.Fatal("Overwhelming Hate does not resolve")
	}
	found := false
	for _, m := range list.Mods() {
		if m.Name() == "Life" && m.Type() == mod.TypeMore && m.Value().Float() == -20 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Overwhelming Hate resolved without Life MORE -20: %v", list.Mods())
	}

	node, ok := legionKeystoneByName("Supreme Decadence")
	if !ok || node.ID != data.LegionKeystones["eternal_keystone_1"].ID {
		t.Fatalf("Supreme Decadence resolved to %q (%v), want eternal_keystone_1", node.ID, ok)
	}
	if control := keystoneModList(env, data.LatestTreeVersion, "Supreme Decadence"); control == nil || len(control.Mods()) == 0 {
		t.Fatal("Supreme Decadence no longer resolves to any mod")
	}
}
