package calculator

import (
	"fmt"
	"sort"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestBarkskinRowsCarryTheirMultiplier: Barkskin's three absent producers.
func TestBarkskinRowsCarryTheirMultiplier(t *testing.T) {
	assertBarkRow(t,
		"Prevent +3% of Suppressed Spell Damage per Bark below maximum",
		"SpellSuppressionEffect", mod.TypeBase, 3, "MissingBarkskinStacks")

	assertBarkRow(t,
		"-25 Damage taken of each Damage Type from Spell Hits per Bark",
		"DamageTakenFromSpells", mod.TypeBase, -25, "BarkskinStacks")

	assertBarkRow(t,
		"Prevent +5% of Suppressed Spell Damage",
		"SpellSuppressionEffect", mod.TypeBase, 5, "")
}

func assertBarkRow(t *testing.T, line, name string, modType mod.Type, value float64, multiplier string) {
	t.Helper()

	mods, extra := parseMod(line, 1)
	if extra != "" {
		t.Errorf("%q: did not parse completely, extra %q -- an unparsed remainder discards the "+
			"WHOLE line, so the node contributes nothing at all", line, extra)
		return
	}
	if len(mods) != 1 {
		t.Errorf("%q: want exactly one mod, got %d", line, len(mods))
		return
	}

	got := mods[0]
	if got.Name() != name || got.Type() != modType {
		t.Errorf("%q: emitted %s/%v, want %s/%v", line, got.Name(), got.Type(), name, modType)
		return
	}
	if v := got.Value(); v == nil || v.Float() != value {
		t.Errorf("%q: emitted value %v, want %v", line, v, value)
		return
	}

	var gotVars []string
	for _, raw := range got.Tags() {
		m, isMult := raw.(*mod.MultiplierTag)
		if !isMult {
			t.Errorf("%q: carries a %T; upstream carries only a Multiplier", line, raw)
			continue
		}
		gotVars = append(gotVars, m.VariableList...)
	}

	if multiplier == "" {
		if len(gotVars) != 0 {
			t.Errorf("%q: carries multipliers %v where the pin's row carries NONE -- the "+
				"neighbouring key has been widened until it swallowed this line", line, gotVars)
		}
		return
	}
	if len(gotVars) != 1 || gotVars[0] != multiplier {
		t.Errorf("%q: multipliers are %v, want exactly [%s]. Without the tag the mod applies its "+
			"BASE value once instead of once per bark -- present, named, and wrong, which every "+
			"producer census scores as covered", line, gotVars, multiplier)
	}
}

// TestBarkskinStacksConfigWritesBothMultipliers: ConfigOptions.lua:294-297.
func TestBarkskinStacksConfigWritesBothMultipliers(t *testing.T) {
	handler, ok := configurations["barkskinStacks"]
	if !ok {
		t.Fatal("no `barkskinStacks` handler; ConfigOptions.lua:294 has one")
	}

	seen := make(map[string]int, len(configOrder))
	for _, name := range configOrder {
		seen[name]++
	}
	if seen["barkskinStacks"] == 0 {
		t.Error("`barkskinStacks` is not in configOrder, so env.go never dispatches it and the " +
			"handler above is dead code that every grep reports as present")
	}
	var dupes []string
	for name, n := range seen {
		if n > 1 {
			dupes = append(dupes, fmt.Sprintf("%s x%d", name, n))
		}
	}
	if len(dupes) > 0 {
		sort.Strings(dupes)
		t.Errorf("configOrder is not a SET: %v. env.go:53 iterates it and looks each name up in "+
			"`configurations`, so a duplicated name applies its handler once per entry — a silent "+
			"double-application, not an error", dupes)
	}

	for name := range configurations {
		if seen[name] == 0 {
			t.Errorf("`%s` has a handler but is not in configOrder, so env.go never calls it", name)
		}
	}

	for _, tc := range []struct{ in, stacks, missing float64 }{
		{0, 0, 0},
		{4, 4, -4},
		{10, 10, -10},
		{25, 10, -10},
	} {
		list := moddb.NewModList()
		handler(tc.in, list, moddb.NewModList())

		stacks := list.Sum(mod.TypeBase, nil, "Multiplier:BarkskinStacks")
		missing := list.Sum(mod.TypeBase, nil, "Multiplier:MissingBarkskinStacks")
		if stacks != tc.stacks || missing != tc.missing {
			t.Errorf("barkskinStacks=%v: got stacks=%v missing=%v, want %v / %v",
				tc.in, stacks, missing, tc.stacks, tc.missing)
		}
	}
}
