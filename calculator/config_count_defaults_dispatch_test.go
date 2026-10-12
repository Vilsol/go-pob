package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestEveryCountDefaultIsDispatched: Every configCountDefaults entry must actually be dispatched.
func TestEveryCountDefaultIsDispatched(t *testing.T) {
	testza.AssertTrue(t, len(configCountDefaults) > 0,
		"REFUSING: configCountDefaults is empty, so this test would pass vacuously")

	inOrder := make(map[string]bool, len(configOrder))
	for _, n := range configOrder {
		inOrder[n] = true
	}

	for name := range configCountDefaults {
		t.Run(name, func(t *testing.T) {
			testza.AssertTrue(t, isConfigDefault(name),
				"isConfigDefault(%q) is false, so env.go never applies its defaultState and "+
					"configDefaultState cannot be reached for it", name)
			testza.AssertTrue(t, inOrder[name],
				"%q is not in configOrder, so env.go:53 never visits it at all", name)
			_, hasHandler := configurations[name]
			testza.AssertTrue(t, hasHandler,
				"%q has no handler in `configurations`, so being dispatched achieves nothing", name)
		})
	}
}
