package calculator2

import (
	"fmt"
	"strings"
	"testing"
)

// A page can reach an item parse before the game data has loaded (or after it failed to load); the parser's
// indices are built by the data's post-load hook, so the call must name that cause rather than dereference nil.
func TestParseModBeforeGameDataNamesTheCause(t *testing.T) {
	saved := specialModIndex
	specialModIndex = nil
	defer func() { specialModIndex = saved }()

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		parseMod("Quarterstaff", 1)
	}()

	if recovered == nil {
		t.Fatal("parseMod returned with no parser indices; it must refuse loudly, not parse to nothing")
	}
	if msg := fmt.Sprint(recovered); !strings.Contains(msg, "before the game data") {
		t.Fatalf("parseMod without game data panicked with %q, which does not name the missing game data", msg)
	}
}
