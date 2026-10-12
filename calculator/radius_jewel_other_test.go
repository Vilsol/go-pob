package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func jewelFuncFor(t *testing.T, line string) radiusJewelFunc {
	t.Helper()
	entry := ParseMod(line, false)
	if strings.Trim(entry.Extra, " ") != "" {
		t.Fatalf("unparsed remainder %q -- the whole line is discarded", entry.Extra)
	}
	if len(entry.ModList) != 1 || entry.ModList[0].Name() != "JewelFunc" {
		t.Fatalf("got %v, want one JewelFunc mod", entry.ModList)
	}
	return entry.ModList[0].Value().List().(jewelFuncEntry).Func
}

// TestNonKeystoneEffectRowIsParameterisedAndSkipsClassStart pins ModParser.lua:6173-6178.
func TestNonKeystoneEffectRowIsParameterisedAndSkipsClassStart(t *testing.T) {
	effectFor := func(t *testing.T, line, nodeType string) float64 {
		t.Helper()
		fn := jewelFuncFor(t, line)
		out := moddb.NewModList()
		fn(&radiusJewelNode{id: "n", nodeType: nodeType, modList: moddb.NewModList()}, out, newRadiusJewelData("Tree:1"))
		return out.Sum(mod.TypeIncrease, nil, "PassiveSkillEffect")
	}

	if got := effectFor(t, "40% increased Effect of non-Keystone Passive Skills in Radius", "Notable"); got != 40 {
		t.Errorf("at 40%%, PassiveSkillEffect = %v, want 40", got)
	}
	if got := effectFor(t, "50% increased Effect of non-Keystone Passive Skills in Radius", "Notable"); got != 50 {
		t.Errorf("at 50%%, PassiveSkillEffect = %v, want 50", got)
	}
	for _, nodeType := range []string{"Keystone", "ClassStart"} {
		if got := effectFor(t, "40% increased Effect of non-Keystone Passive Skills in Radius", nodeType); got != 0 {
			t.Errorf("a %s node got PassiveSkillEffect %v, want 0", nodeType, got)
		}
	}
}

// TestGrantUnallocatedBonusesFiltersConnectedTo pins ModParser.lua:6362-6392.
func TestGrantUnallocatedBonusesFiltersConnectedTo(t *testing.T) {
	for _, c := range []struct {
		line     string
		nodeType string
		other    string
	}{
		{"Grants all bonuses of Unallocated Small Passive Skills in Radius", "Normal", "Notable"},
		{"Grants all bonuses of Unallocated Notable Passive Skills in Radius", "Notable", "Normal"},
	} {
		t.Run(c.nodeType, func(t *testing.T) {
			fn := jewelFuncFor(t, c.line)
			d := newRadiusJewelData("Tree:1")

			work := moddb.NewModList()
			work.AddMod(mod.NewFloat("Life", mod.TypeBase, 20))
			work.AddMod(mod.NewFlag("Condition:ConnectedToDuelistStart", true))
			fn(&radiusJewelNode{id: "n", nodeType: c.nodeType, modList: moddb.NewModList()}, work, d)

			otherWork := moddb.NewModList()
			otherWork.AddMod(mod.NewFloat("Life", mod.TypeBase, 500))
			fn(&radiusJewelNode{id: "o", nodeType: c.other, modList: moddb.NewModList()}, otherWork, d)

			out := moddb.NewModList()
			fn(nil, out, d)

			if got := out.Sum(mod.TypeBase, nil, "Life"); got != 20 {
				t.Errorf("Life = %v, want 20 -- 520 means the wrong node type was collected, 0 means the filter ate everything", got)
			}
			if out.Flag(nil, "Condition:ConnectedToDuelistStart") {
				t.Error("a Condition:ConnectedTo mod survived; an unallocated node is not on the character's path")
			}
		})
	}
}

// TestAllocatedNotableGrantsNothing pins ModParser.lua:6197, the NOTABLE twin of a row that was ported alone.
func TestAllocatedNotableGrantsNothing(t *testing.T) {
	for _, c := range []struct {
		line     string
		fires    string
		notFires string
	}{
		{"Allocated Notable Passive Skills in Radius grant nothing", "Notable", "Normal"},
		{"Allocated Small Passive Skills in Radius grant nothing", "Normal", "Notable"},
	} {
		t.Run(c.fires, func(t *testing.T) {
			fn := jewelFuncFor(t, c.line)
			for nodeType, want := range map[string]bool{c.fires: true, c.notFires: false} {
				out := moddb.NewModList()
				fn(&radiusJewelNode{id: "n", nodeType: nodeType, modList: moddb.NewModList()}, out, newRadiusJewelData("Tree:1"))
				if got := out.Flag(nil, "AllocatedPassiveSkillHasNoEffect"); got != want {
					t.Errorf("%s node: flag %v, want %v", nodeType, got, want)
				}
			}
		})
	}
}
