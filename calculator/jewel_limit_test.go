package calculator

import (
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

// TestJewelLimitDisablesTheExtraSocket pins CalcSetup.lua:760-771.
func TestJewelLimitDisablesTheExtraSocket(t *testing.T) {
	requireGameData(t, "3.29")

	const watchersEye = "Rarity: UNIQUE\nWatcher's Eye\nPrismatic Jewel\nLimited to: 1\n" +
		"Variant: Anger\n+5% to Critical Strike Multiplier\n"
	const gloriousVanity = "Rarity: UNIQUE\nGlorious Vanity\nTimeless Jewel\nLimited to: 1 Historic\n" +
		"Variant: Doryani\nBathed in the blood of 100 sacrificed in the name of Xibaqua\n"
	const lethalPride = "Rarity: UNIQUE\nLethal Pride\nTimeless Jewel\nLimited to: 1 Historic\n" +
		"Variant: Kaom\nCommanded leadership over 10000 warriors under Kaom\n"

	env := func(items map[int]string, sockets []pob.Socket, ignore bool) *Environment {
		list := make([]pob.Item, 0, len(items))
		for id, text := range items {
			list = append(list, pob.Item{ID: id, Text: text})
		}
		cfg := pob.Config{}
		if ignore {
			yes := true
			cfg.Inputs = []pob.Input{{Name: "ignoreJewelLimits", Boolean: &yes}}
		}
		alloc := make(map[string]data.Node, len(sockets))
		for _, s := range sockets {
			alloc[strconv.Itoa(s.NodeID)] = data.Node{}
		}
		return &Environment{
			Build: &pob.PathOfBuilding{
				Tree:   pob.Tree{ActiveSpec: 1, Specs: []pob.Spec{{Sockets: sockets}}},
				Items:  pob.Items{Items: list},
				Config: cfg,
			},
			AllocatedNodes: alloc,
			Cache:          NewEnvironmentCache(),
		}
	}

	t.Run("the second jewel over a shared limit is disabled", func(t *testing.T) {
		e := env(map[int]string{1: watchersEye, 2: watchersEye},
			[]pob.Socket{{ItemID: 1, NodeID: 500}, {ItemID: 2, NodeID: 100}}, false)

		got := limitDisabledJewelSockets(e)
		if len(got) != 1 {
			t.Fatalf("disabled %d socket(s), want 1", len(got))
		}
		if !got[500] {
			t.Errorf("disabled %v, want node 500; the survivor is the LOWEST node id, not the "+
				"first <Socket> element", got)
		}
	})

	t.Run("two DIFFERENT timeless jewels still share one limit", func(t *testing.T) {
		e := env(map[int]string{1: gloriousVanity, 2: lethalPride},
			[]pob.Socket{{ItemID: 1, NodeID: 100}, {ItemID: 2, NodeID: 500}}, false)

		got := limitDisabledJewelSockets(e)
		if len(got) != 1 || !got[500] {
			t.Errorf("disabled %v, want exactly node 500: CalcSetup.lua:761 keys a Timeless jewel "+
				"as \"Historic\" rather than by title, so Glorious Vanity plus Lethal Pride is two "+
				"of a limit of one", got)
		}
	})

	t.Run("different titles do not share a limit", func(t *testing.T) {
		e := env(map[int]string{1: watchersEye, 2: gloriousVanity},
			[]pob.Socket{{ItemID: 1, NodeID: 100}, {ItemID: 2, NodeID: 500}}, false)

		if got := limitDisabledJewelSockets(e); len(got) != 0 {
			t.Errorf("disabled %v, want none; a Watcher's Eye and a timeless jewel are different "+
				"limit keys and this is the control against disabling on count alone", got)
		}
	})

	t.Run("ignoreJewelLimits disables the whole check", func(t *testing.T) {
		e := env(map[int]string{1: watchersEye, 2: watchersEye},
			[]pob.Socket{{ItemID: 1, NodeID: 100}, {ItemID: 2, NodeID: 500}}, true)

		if got := limitDisabledJewelSockets(e); len(got) != 0 {
			t.Errorf("disabled %v with ignoreJewelLimits set, want none (CalcSetup.lua:760); 12 "+
				"corpus builds set this option", got)
		}
	})

	t.Run("an unallocated socket never reaches the check", func(t *testing.T) {
		e := env(map[int]string{1: watchersEye, 2: watchersEye},
			[]pob.Socket{{ItemID: 1, NodeID: 100}, {ItemID: 2, NodeID: 500}}, false)
		delete(e.AllocatedNodes, "500")

		if got := limitDisabledJewelSockets(e); len(got) != 0 {
			t.Errorf("disabled %v, want none: CalcSetup.lua:746 skips an unallocated socket "+
				"BEFORE the limit check, so it must not consume the allowance either", got)
		}
	})
}
