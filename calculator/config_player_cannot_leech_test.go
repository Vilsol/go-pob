package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestPlayerCannotLeechIsInert pins the user decision of 2026-10-11: PoB has no such option and ignores the input.
func TestPlayerCannotLeechIsInert(t *testing.T) {
	apply, ok := configurations["playerCannotLeech"]
	if !ok {
		t.Fatal("playerCannotLeech has no handler; the go-pob-only option must stay a row")
	}
	player, enemy := moddb.NewModList(), moddb.NewModList()
	apply(true, player, enemy)
	if n := len(player.Mods()) + len(enemy.Mods()); n != 0 {
		t.Errorf("checked playerCannotLeech added %d mods; PoB applies nothing", n)
	}
}
