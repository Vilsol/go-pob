package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// Expected values are what each pin's own ModDB/ModStore returns for the same mod and cfg under luajit.

type poe2TestGem struct {
	tags             map[string]bool
	statSetBaseFlags map[string]bool
}

func (g *poe2TestGem) GemTags() map[string]bool {
	return g.tags
}

type poe1TestGrantedEffect struct {
	baseFlags map[string]bool
}

func (g *poe1TestGrantedEffect) BaseFlags() map[string]bool {
	return g.baseFlags
}

func poe2TestRules() *Rules {
	return &Rules{
		LimitNeg:       true,
		SumWholeSource: true,
		BaseFlags: func(cfg *ListCfg) map[string]bool {
			if cfg == nil {
				return nil
			}
			if gem, ok := cfg.GemData.(*poe2TestGem); ok && gem.statSetBaseFlags != nil {
				return gem.statSetBaseFlags
			}
			return cfg.BaseFlags
		},
	}
}

func dbFor(rules *Rules, mods ...mod.Mod) *ModDB {
	db := NewModDB()
	db.Rules = rules
	for _, m := range mods {
		db.AddMod(m)
	}
	return db
}

func TestRulesSumWholeSource(t *testing.T) {
	cases := []struct {
		name       string
		cfgSource  string
		poe1, poe2 float64
	}{
		{"whole source", "Item:5:Foo", 0, 10},
		{"prefix", "Item", 10, 10},
		{"other", "Tree", 0, 0},
	}
	for _, c := range cases {
		for _, game := range []struct {
			rules *Rules
			want  float64
		}{{nil, c.poe1}, {poe2TestRules(), c.poe2}} {
			db := dbFor(game.rules, mod.NewFloat("Life", mod.TypeBase, 10).Source("Item:5:Foo"))
			testza.AssertEqual(t, game.want, db.Sum(mod.TypeBase, srcCfg(c.cfgSource), "Life"), c.name)
		}
	}

	t.Run("only ModDB Sum takes the whole source", func(t *testing.T) {
		db := dbFor(poe2TestRules(), mod.NewFloat("Life", mod.TypeMore, 10).Source("Item:5:Foo"))
		testza.AssertEqual(t, float64(1), db.More(srcCfg("Item:5:Foo"), "Life"))

		list := NewModList()
		list.Rules = poe2TestRules()
		list.AddMod(mod.NewFloat("Life", mod.TypeBase, 10).Source("Item:5:Foo"))
		testza.AssertEqual(t, float64(0), list.Sum(mod.TypeBase, srcCfg("Item:5:Foo"), "Life"))
	})

	t.Run("a child list inherits its parent's rules", func(t *testing.T) {
		parent := dbFor(poe2TestRules(), mod.NewFloat("Life", mod.TypeBase, 10).Source("Item:5:Foo"))
		child := NewModList()
		child.Parent = parent
		testza.AssertEqual(t, float64(10), child.Sum(mod.TypeBase, srcCfg("Item:5:Foo"), "Life"))
	})
}

func TestRulesLimitNeg(t *testing.T) {
	cases := []struct {
		name       string
		value      float64
		neg        bool
		poe1, poe2 float64
	}{
		{"neg below", -100, true, -100, -90},
		{"neg above", 100, true, 90, 100},
		{"plain above", 100, false, 90, 90},
	}
	for _, c := range cases {
		for _, game := range []struct {
			rules *Rules
			want  float64
		}{{nil, c.poe1}, {poe2TestRules(), c.poe2}} {
			db := dbFor(game.rules, mod.NewFloat("X", mod.TypeIncrease, c.value).Tag(mod.Limit(90).Neg(c.neg)))
			testza.AssertEqual(t, game.want, db.Sum(mod.TypeIncrease, nil, "X"), c.name)
		}
	}
}

func TestRulesBaseFlags(t *testing.T) {
	duration := map[string]bool{"duration": true}
	cases := []struct {
		name       string
		cfg        *ListCfg
		neg        bool
		poe1, poe2 float64
	}{
		{"granted effect", &ListCfg{SkillGrantedEffect: &poe1TestGrantedEffect{baseFlags: duration}}, false, 5, 0},
		{"gem stat set", &ListCfg{GemData: &poe2TestGem{statSetBaseFlags: duration}}, false, 0, 5},
		{"cfg baseFlags", &ListCfg{BaseFlags: duration}, false, 5, 5},
		{"none", &ListCfg{}, false, 0, 0},
		{"neg granted effect", &ListCfg{SkillGrantedEffect: &poe1TestGrantedEffect{baseFlags: duration}}, true, 0, 5},
		{"neg gem stat set", &ListCfg{GemData: &poe2TestGem{statSetBaseFlags: duration}}, true, 5, 0},
	}
	for _, c := range cases {
		for _, game := range []struct {
			rules *Rules
			want  float64
		}{{nil, c.poe1}, {poe2TestRules(), c.poe2}} {
			db := dbFor(game.rules, mod.NewFloat("X", mod.TypeBase, 5).Tag(mod.BaseFlag("duration").Neg(c.neg)))
			testza.AssertEqual(t, game.want, db.Sum(mod.TypeBase, c.cfg, "X"), c.name)
		}
	}
}

func TestGemTagTag(t *testing.T) {
	fire := &ListCfg{GemData: &poe2TestGem{tags: map[string]bool{"fire": true}}}
	cases := []struct {
		name string
		tag  *mod.GemTagTag
		cfg  *ListCfg
		want float64
	}{
		{"match", mod.GemTag("Fire"), fire, 5},
		{"miss", mod.GemTag("Cold"), fire, 0},
		{"list", mod.GemTag("Cold", "Fire"), fire, 5},
		{"neg", mod.GemTag("Fire").Neg(true), fire, 0},
		{"no gem", mod.GemTag("Fire"), &ListCfg{}, 0},
	}
	for _, c := range cases {
		db := dbFor(nil, mod.NewFloat("X", mod.TypeBase, 5).Tag(c.tag))
		testza.AssertEqual(t, c.want, db.Sum(mod.TypeBase, c.cfg, "X"), c.name)
	}
}

func TestSkillNamePartialMatch(t *testing.T) {
	cfg := &ListCfg{SkillName: "Spectre: Zombie", SummonSkillName: "Raise Spectre"}
	cases := []struct {
		name string
		tag  *mod.SkillNameTag
		want float64
	}{
		{"list", mod.SkillName("Spectre", "Companion").PartialMatch(true), 5},
		{"single", mod.SkillName("spectre").PartialMatch(true), 5},
		{"off", mod.SkillName("Spectre"), 0},
		{"summon neg", mod.SkillName("Spectre", "Companion").PartialMatch(true).SummonSkill(true).Neg(true), 0},
	}
	for _, c := range cases {
		db := dbFor(nil, mod.NewFloat("X", mod.TypeBase, 5).Tag(c.tag))
		testza.AssertEqual(t, c.want, db.Sum(mod.TypeBase, cfg, "X"), c.name)
	}
}

func TestSocketedInSlotType(t *testing.T) {
	cfg := &ListCfg{SlotName: "Helmet"}
	cases := []struct {
		name string
		tag  *mod.SocketedInTag
		want float64
	}{
		{"slot type only", mod.SocketedIn("").SlotType("Weapon"), 5},
		{"slot type wins over slot name", mod.SocketedIn("Weapon 1").SlotType("Weapon"), 5},
		{"neg", mod.SocketedIn("").SlotType("Weapon").Neg(true), 0},
	}
	for _, c := range cases {
		db := dbFor(nil, mod.NewFloat("X", mod.TypeBase, 5).Tag(c.tag))
		testza.AssertEqual(t, c.want, db.Sum(mod.TypeBase, cfg, "X"), c.name)
	}
}

func TestSumPositiveNegativeValues(t *testing.T) {
	mods := []mod.Mod{
		mod.NewFloat("X", mod.TypeIncrease, 30).Source("A"),
		mod.NewFloat("X", mod.TypeIncrease, -20).Source("B"),
		mod.NewFloat("X", mod.TypeIncrease, 15).Source("C"),
		mod.NewFloat("Y", mod.TypeIncrease, 99).Source("D"),
	}
	db := dbFor(nil, mods...)
	testza.AssertEqual(t, float64(45), db.SumPositiveValues(mod.TypeIncrease, nil, "X", "Y"))
	testza.AssertEqual(t, float64(-20), db.SumNegativeValues(mod.TypeIncrease, nil, "X", "Y"))

	list := NewModList()
	for _, m := range mods {
		list.AddMod(m)
	}
	testza.AssertEqual(t, float64(45), list.SumPositiveValues(mod.TypeIncrease, nil, "X", "Y"))
	testza.AssertEqual(t, float64(-20), list.SumNegativeValues(mod.TypeIncrease, nil, "X", "Y"))
}

func TestGetConditionOverrideCond(t *testing.T) {
	db := NewModDB()
	db.Conditions["Foo"] = true

	got, _ := db.GetCondition("Foo", &ListCfg{OverrideCond: map[string]bool{"Foo": false}}, false)
	testza.AssertFalse(t, got)

	got, _ = db.GetCondition("Bar", &ListCfg{OverrideCond: map[string]bool{"Bar": true}}, false)
	testza.AssertTrue(t, got)

	got, _ = db.GetCondition("Foo", &ListCfg{OverrideCond: map[string]bool{"Bar": false}}, false)
	testza.AssertTrue(t, got)
}
