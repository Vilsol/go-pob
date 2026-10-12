package calculator

import (
	"testing"
)

// TestBlockChanceMatchesThePin pins CalcDefence.lua:686-717.
func TestBlockChanceMatchesThePin(t *testing.T) {
	t.Run("the total is rounded", func(t *testing.T) {
		assertBuildStats(t, "0062-occultist-crackling-lance.xml", []string{"BlockChance"}, nil)
	})
}
