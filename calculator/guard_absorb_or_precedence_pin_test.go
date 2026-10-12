package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGuardAbsorbFixupKeepsThePinsOrPrecedence pins the ONE thing calc_defence.go's TotalHitPool guard fix-up cannot prove for itself: that upstream still writes.
//
//	local GuardAbsorbRate = output["sharedGuardAbsorbRate"] or 0 + output[damageType.."GuardAbsorbRate"] or 0
func TestGuardAbsorbFixupKeepsThePinsOrPrecedence(t *testing.T) {
	src := pinnedPoBSrcOrSkip(t)
	path := filepath.Join(src, "Modules", "CalcDefence.lua")
	raw, err := os.ReadFile(path) //nolint:gosec // a path under the pinned checkout, resolved above
	if err != nil {
		t.Fatalf("REFUSING: cannot read %s: %v", path, err)
	}
	lua := string(raw)

	for _, want := range []string{
		`local GuardAbsorbRate = output["sharedGuardAbsorbRate"] or 0 + output[damageType.."GuardAbsorbRate"] or 0`,
		`local GuardAbsorb = output["sharedGuardAbsorb"] or 0 + output[damageType.."GuardAbsorb"] or 0`,
	} {
		if !strings.Contains(lua, want) {
			t.Errorf("the pin no longer contains\n\t%s\nin CalcDefence.lua. Either the line moved "+
				"(re-quote it) or upstream PARENTHESISED it, in which case the per-type term is now "+
				"LIVE and calc_defence.go's guard fix-up must add it back — it currently reads the "+
				"shared key alone, which is what the unparenthesised `or` actually evaluates to.",
				want)
		}
	}

	if !strings.Contains(lua, `if guard[damageType] > 0 then`) || !strings.Contains(lua, `if guard.shared > 0 then`) {
		t.Errorf("REFUSING: the two separate guard-pool layers in reducePoolsByDamage are gone from " +
			"the pin, so this test can no longer tell 'upstream ignores the per-type guard' from " +
			"'upstream reads it somewhere else' — re-derive before trusting the assertion above")
	}
}
