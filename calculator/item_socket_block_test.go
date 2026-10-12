package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestItemSocketBlock: Item.lua:2588-2635 reshapes self.sockets from the item's own local mods.
//
//pob:block:src/Classes/Item.lua:2588-2635
func TestItemSocketBlock(t *testing.T) {
	const gloves = "Rarity: UNIQUE\nTest Gloves\nSorcerer Gloves\n"

	t.Run("a base with no socket mod keeps its default fill", func(t *testing.T) {
		item := ParseItem(gloves + "Item Level: 84\n")
		if item.Base == nil || item.Base.SocketLimit == 0 {
			t.Fatalf("REFUSING: Sorcerer Gloves resolved no base with a socket limit (%v)", item.Base)
		}
		if len(item.Sockets) != item.Base.SocketLimit {
			t.Errorf("sockets = %d, want the base limit %d", len(item.Sockets), item.Base.SocketLimit)
		}
	})

	t.Run("NoSockets wipes every socket", func(t *testing.T) {
		item := ParseItem(gloves + "Item Level: 84\nHas no Sockets\n")
		if len(item.Sockets) != 0 {
			t.Errorf("sockets = %d, want 0", len(item.Sockets))
		}
		if item.AbyssalSocketCount != 0 {
			t.Errorf("AbyssalSocketCount = %v, want 0", item.AbyssalSocketCount)
		}
		if n := countModsNamed(item.Mods, "NoSockets"); n != 0 {
			t.Errorf("NoSockets survived into the global list %d times; calcLocal must consume it", n)
		}
		if n := countMultipliersContaining(item.Mods, "SocketIn"); n != 0 {
			t.Errorf("%d socket multipliers emitted for an item with no sockets", n)
		}
	})

	t.Run("SocketCount forces the socket count down", func(t *testing.T) {
		item := ParseItem(gloves + "Item Level: 84\nHas 1 Socket\n")
		if len(item.Sockets) != 1 {
			t.Errorf("sockets = %d, want 1", len(item.Sockets))
		}
		if n := countModsNamed(item.Mods, "SocketCount"); n != 0 {
			t.Errorf("SocketCount survived into the global list %d times; calcLocal must consume it", n)
		}
	})
}

func countModsNamed(mods []mod.Mod, name string) int {
	n := 0
	for _, m := range mods {
		if m.Name() == name {
			n++
		}
	}
	return n
}

func countMultipliersContaining(mods []mod.Mod, fragment string) int {
	n := 0
	for _, m := range mods {
		if len(m.Name()) > 11 && m.Name()[:11] == "Multiplier:" && containsStr(m.Name(), fragment) {
			n++
		}
	}
	return n
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
