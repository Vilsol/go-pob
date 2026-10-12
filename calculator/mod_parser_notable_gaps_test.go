package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestNotableParserGapRows: The three anointable notables' lines the pin parses.
func TestNotableParserGapRows(t *testing.T) {
	for _, c := range []struct {
		line, name string
		typ        mod.Type
		value      float64
	}{
		{"+8 to Minimum Rage", "MinimumRage", mod.TypeBase, 8},
		{"Create Fungal Ground instead of Consecrated Ground", "Condition:CreateFungalGround", mod.TypeFlag, 1},
		{"Tinctures have 40% increased effect while at or above 10 stacks of Mana Burn", "TinctureEffect", mod.TypeIncrease, 40},
	} {
		e := ParseMod(c.line, false)
		if e == nil || strings.TrimSpace(e.Extra) != "" || len(e.ModList) != 1 {
			t.Errorf("%q: %+v", c.line, e)
			continue
		}
		m := e.ModList[0]
		v := m.Value().Float()
		if m.Type() == mod.TypeFlag && m.Value().Flag() {
			v = 1
		}
		if m.Name() != c.name || m.Type() != c.typ || v != c.value {
			t.Errorf("%q: %s %s %v", c.line, m.Name(), m.Type(), v)
		}
	}

	e := ParseMod("Tinctures have 40% increased effect while at or above 10 stacks of Mana Burn", false)
	if e == nil || len(e.ModList) != 1 {
		t.Fatal("no mod")
	}
	tags := e.ModList[0].Tags()
	tag, ok := tags[0].(*mod.MultiplierThresholdTag)
	if len(tags) != 1 || !ok || !slices.Equal(tag.VariableList, []string{"ManaBurnStacks", "WeepingWoundsStacks"}) || tag.TagThreshold == nil || *tag.TagThreshold != 10 {
		t.Fatalf("tags %#v", tags)
	}

	db := moddb.NewModDB()
	db.AddMod(e.ModList[0])
	db.AddMod(mod.NewFloat("Multiplier:ManaBurnStacks", mod.TypeBase, 6))
	if got := db.Sum(mod.TypeIncrease, nil, "TinctureEffect"); got != 0 {
		t.Errorf("6 stacks: %v", got)
	}
	db.AddMod(mod.NewFloat("Multiplier:WeepingWoundsStacks", mod.TypeBase, 4))
	if got := db.Sum(mod.TypeIncrease, nil, "TinctureEffect"); got != 40 {
		t.Errorf("6 + 4 stacks: %v", got)
	}
}
