package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestNodeStatParsePassIsNotIndex: PoB's PassiveTree:ProcessStats calls modLib.parseMod(line).
func TestNodeStatParsePassIsNotIndex(t *testing.T) {
	name := "Blizzard Caller"
	node := data.Node{
		Name: &name,
		Stats: []string{
			"40% increased Frostbite Curse Effect",
			"15% increased Critical Strike Chance",
		},
	}

	env := &Environment{}
	ml := buildModListForNode(env, "25938", node)

	testza.AssertEqual(t, 40.0, ml.Sum(mod.TypeIncrease, &moddb.ListCfg{SkillName: "Frostbite"}, "CurseEffect"), "first-line skill-name mod must parse")

	var found *mod.SkillNameTag
	for _, m := range ml.Mods() {
		if m.Name() != "CurseEffect" {
			continue
		}
		for _, tg := range m.Tags() {
			if sn, ok := tg.(*mod.SkillNameTag); ok {
				found = sn
			}
		}
	}
	if testza.AssertNotNil(t, found, "CurseEffect mod must carry a SkillName tag"); found != nil {
		testza.AssertEqual(t, []string{"Frostbite"}, found.SkillNameList)
	}
	testza.AssertEqual(t, 15.0, ml.Sum(mod.TypeIncrease, nil, "CritChance"))
}
