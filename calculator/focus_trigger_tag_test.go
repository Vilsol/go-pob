package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestFocusTriggerRowsCarryOnlyTheSocketedInTag pins ModParser.lua:3153-3154.
func TestFocusTriggerRowsCarryOnlyTheSocketedInTag(t *testing.T) {
	tagTypes := func(t *testing.T, line string, wantMods int) [][]mod.Type {
		t.Helper()
		e := ParseMod(line, false)
		if e == nil {
			t.Fatalf("ParseMod(%q) returned nothing", line)
		}
		if e.Extra != "" {
			t.Fatalf("%q left %q unparsed — an unparsed remainder discards the WHOLE line",
				line, e.Extra)
		}
		if len(e.ModList) != wantMods {
			t.Fatalf("%q produced %d mods, want %d", line, len(e.ModList), wantMods)
		}
		out := make([][]mod.Type, 0, len(e.ModList))
		for _, m := range e.ModList {
			types := make([]mod.Type, 0, len(m.Tags()))
			for _, tag := range m.Tags() {
				types = append(types, tag.Type())
			}
			out = append(out, types)
		}
		return out
	}

	for _, line := range []string{
		"Trigger Socketed Spells when you Focus",
		"Trigger Socketed Spells when you Focus, with a 0.25 second Cooldown",
	} {
		t.Run(line, func(t *testing.T) {
			got := tagTypes(t, line, 1)
			testza.AssertEqual(t, []mod.Type{mod.TypeSocketedIn}, got[0],
				"ModParser.lua:3153-3154 emit this mod with the SocketedIn tag alone. A "+
					"Condition tag here gates INSTALLATION of the support, not its trigger rate, "+
					"so the support is never socketed and the trigger can never fire")
		})
	}

	t.Run("neighbour keeps its Focused condition", func(t *testing.T) {
		got := tagTypes(t, "Shock nearby Enemies for 4 Seconds when you Focus", 2)
		for i, tags := range got {
			testza.AssertContains(t, tags, mod.TypeCondition,
				"ModParser.lua:3728-3730 DO tag this row with Condition Focused; mod %d lost it, "+
					"which is the over-application half of the same defect", i)
		}
	})
}
