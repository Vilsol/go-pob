package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestWhileInOffHandParsesAsSlotNumber: ModParser.lua:1587's key is `whi?l?en?
func TestWhileInOffHandParsesAsSlotNumber(t *testing.T) {
	for _, line := range []string{
		"+38% Chance to Block Spell Damage while in Off Hand",
		"+38% Chance to Block Spell Damage when in Off Hand",
	} {
		mods, extra := parseMod(line, 1)
		if extra != "" {
			t.Errorf("%q: did not parse completely, extra %q", line, extra)
			continue
		}
		if len(mods) != 1 {
			t.Errorf("%q: want exactly one mod, got %d", line, len(mods))
			continue
		}
		got := mods[0]
		if got.Name() != "SpellBlockChance" || got.Type() != mod.TypeBase {
			t.Errorf("%q: emitted %s/%v, want SpellBlockChance/BASE", line, got.Name(), got.Type())
			continue
		}
		if v := got.Value(); v == nil || v.Float() != 38 {
			t.Errorf("%q: emitted value %v, want 38", line, v)
			continue
		}
		tags := got.Tags()
		if len(tags) != 1 {
			t.Errorf("%q: carries %d tags, want exactly 1 (SlotNumber 2)", line, len(tags))
			continue
		}
		slot, ok := tags[0].(*mod.SlotNumberTag)
		if !ok {
			t.Errorf("%q: tag is %T, want *mod.SlotNumberTag — `in off hand` (InSlot) is a "+
				"DIFFERENT tag and reaches a different consumer", line, tags[0])
			continue
		}
		if slot.N != 2 {
			t.Errorf("%q: SlotNumber is %d, want 2", line, slot.N)
		}
	}
}

// TestSlotNumberKeysMatchThePinsAsymmetry: THE MAIN-HAND ASYMMETRY IS UPSTREAM'S, NOT AN OVERSIGHT HERE.
func TestSlotNumberKeysMatchThePinsAsymmetry(t *testing.T) {
	src := pinnedPoBSrcOrSkip(t)
	body, err := os.ReadFile(filepath.Join(src, "Modules", "ModParser.lua"))
	if err != nil {
		t.Skipf("NOT-RUN: cannot read the pinned ModParser.lua: %v", err)
	}

	row := regexp.MustCompile(`\["([^"]*in (?:main|off) hand)"\] = \{ tag = \{ type = "SlotNumber", num = (\d) \} \}`)
	found := map[string]string{}
	for _, m := range row.FindAllStringSubmatch(string(body), -1) {
		found[m[1]] = m[2]
	}
	if len(found) == 0 {
		t.Fatal("REFUSING: scanned the pinned ModParser.lua and found no SlotNumber hand rows at " +
			"all — the entry shape moved, so an empty scan here is the CHECKER failing, not the pin")
	}

	want := map[string]string{
		"when in main hand":     "1",
		"whi?l?en? in off hand": "2",
	}
	for key, num := range want {
		if got, ok := found[key]; !ok || got != num {
			t.Errorf("pinned ModParser.lua no longer carries %q -> SlotNumber %s (found %v). "+
				"If upstream symmetrised these keys, mod_parser.go must follow; if it narrowed "+
				"them, this port is now over-matching", key, num, found)
		}
	}
	if len(found) != len(want) {
		t.Errorf("pinned ModParser.lua carries %d SlotNumber hand rows, this port mirrors %d: %v",
			len(found), len(want), found)
	}

	for key := range want {
		if _, ok := modTagList[key]; !ok {
			t.Errorf("modTagList has no %q; the pin does", key)
		}
	}
	if _, ok := modTagList["while in main hand"]; ok {
		t.Error("modTagList has a `while in main hand` key the pin does not — that is a " +
			"divergence we introduced, not a fix")
	}

	if _, extra := parseMod("+38% Chance to Block Spell Damage while in Main Hand", 1); !strings.Contains(extra, "while") {
		t.Error("`while in Main Hand` now parses; the pin drops it, and porting fidelity means " +
			"porting the ABSENCES too")
	}
}
