package calculator2

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// The "also grant" rows match only the node type they name and parse the granted line with
// PoB-PoE2's parser; the granted mods equal the shipped ModCache row for the same line.
func TestRadiusAlsoGrantJewelFunc(t *testing.T) {
	mods, extra := ParseMod("Notable Passive Skills in Radius also grant 10% increased Critical Hit Chance")
	testza.AssertEqual(t, "", extra)
	testza.AssertLen(t, mods, 1)
	entry, ok := mods[0].Value().List().(jewelFuncEntry)
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, "Other", entry.Type)

	want, wantExtra := ModCache("10% increased Critical Hit Chance")
	testza.AssertEqual(t, "", wantExtra)
	testza.AssertLen(t, want, 1)

	d := newRadiusJewelData("Tree:1")
	for _, tc := range []struct {
		node *radiusJewelNode
		n    int
	}{
		{&radiusJewelNode{nodeType: "Notable", modList: moddb.NewModList()}, 1},
		{&radiusJewelNode{nodeType: "Normal", modList: moddb.NewModList()}, 0},
		{nil, 0},
	} {
		out := moddb.NewModList()
		entry.Func(tc.node, out, d)
		testza.AssertLen(t, out.Mods(), tc.n)
		for _, m := range out.Mods() {
			testza.AssertEqual(t, want[0].Name(), m.Name())
			testza.AssertEqual(t, want[0].Type(), m.Type())
			testza.AssertEqual(t, want[0].Value().Float(), m.Value().Float())
			testza.AssertEqual(t, mod.Source("Tree:1"), m.GetSource())
		}
	}
}

// A text capture reaches a special row's num, and a text form capture the form value, as Lua's
// tonumber nil: no float parse is attempted on it. The last three lines are rows of the shipped ModCache.
func TestParseModTextCapturesAreNotNumbers(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	initializeParser()
	for _, line := range []string{
		"Passives in Radius of Vaal Pact can be Allocated without being connected to your tree",
		"Gain additional Stun Threshold equal to 12% of maximum Energy Shield",
		"You are considered on Low Life while at 75% of maximum Life or below instead",
		"Gain Physical Thorns damage equal to 5% of Item Armour on Equipped Body Armour",
	} {
		parseMod(line, 1)
	}
	testza.AssertFalse(t, strings.Contains(buf.String(), "failed to parse"), buf.String())

	mods, extra := ParseMod("Gain additional Stun Threshold equal to 12% of maximum Energy Shield")
	want, wantExtra := ModCache("Gain additional Stun Threshold equal to 12% of maximum Energy Shield")
	testza.AssertEqual(t, wantExtra, extra)
	testza.AssertLen(t, mods, len(want))
	testza.AssertEqual(t, want[0].Name(), mods[0].Name())
	testza.AssertEqual(t, want[0].Value().Float(), mods[0].Value().Float())
}
