package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// Expected values are what each pin's own ModDB/ModList/ModStore returns for the same input under luajit.

func poe2HookRules() *Rules {
	r := poe2TestRules()
	r.ReplaceIgnoresReplaced = true
	r.ScaleLevelFloor = true
	r.ManaUnreservedNegativeDead = true
	r.HighPrecisionMods = map[string]map[mod.Type]int{
		"WardRegenPercent":      {mod.TypeBase: 2},
		"ReservationMultiplier": {mod.TypeMore: 4},
	}
	return r
}

func TestRulesReplaceModTwice(t *testing.T) {
	for _, game := range []struct {
		rules *Rules
		want  float64
	}{{nil, 3}, {poe2HookRules(), 22}} {
		db := dbFor(game.rules,
			mod.NewFloat("Life", mod.TypeBase, 10).Source("S"),
			mod.NewFloat("Life", mod.TypeBase, 20).Source("S"))
		db.ReplaceMod(mod.NewFloat("Life", mod.TypeBase, 1).Source("S"))
		db.ReplaceMod(mod.NewFloat("Life", mod.TypeBase, 2).Source("S"))
		testza.AssertEqual(t, game.want, db.Sum(mod.TypeBase, nil, "Life"))
		testza.AssertEqual(t, 2, len(db.Mods["Life"]))
	}
}

func TestRulesScaleAddModKeyOfScaledMod(t *testing.T) {
	list := NewModList()
	list.Rules = poe2HookRules()
	list.ScaleAddMod(mod.NewList("GemProperty", mod.GemProperty{Key: "level", Value: 3}).Source("S"), 1.5, false)
	list.ScaleAddMod(mod.NewList("GemProperty", &mod.GemProperty{Key: "quality", Value: 3}).Source("S"), 1.555, false)
	got := list.List(nil, "GemProperty")
	testza.AssertEqual(t, 2, len(got))
	testza.AssertEqual(t, float64(4), got[0].(mod.GemProperty).Value)
	testza.AssertEqual(t, 4.67, got[1].(*mod.GemProperty).Value)
}

func TestRulesScaleAddModRoundToNearest(t *testing.T) {
	list := NewModList()
	list.Rules = poe2HookRules()
	list.ScaleAddMod(mod.NewFloat("Life", mod.TypeBase, 5).Source("S"), 1.5, true)
	list.ScaleAddMod(mod.NewFloat("Mana", mod.TypeBase, -5).Source("S"), 1.5, true)
	list.ScaleAddMod(mod.NewFloat("Armour", mod.TypeBase, 5).Source("S"), 1.5, false)
	testza.AssertEqual(t, float64(8), list.Sum(mod.TypeBase, nil, "Life"))
	testza.AssertEqual(t, float64(-8), list.Sum(mod.TypeBase, nil, "Mana"))
	testza.AssertEqual(t, float64(7), list.Sum(mod.TypeBase, nil, "Armour"))
}

func TestRulesHighPrecisionMods(t *testing.T) {
	for _, game := range []struct {
		rules           *Rules
		ward, tincture  float64
		moreReservation float64
	}{{nil, 1, 1.23, 1.16}, {poe2HookRules(), 1.23, 1, 1.1608}} {
		list := NewModList()
		list.Rules = game.rules
		list.ScaleAddList(listOf(
			mod.NewFloat("WardRegenPercent", mod.TypeBase, 1).Source("S"),
			mod.NewFloat("LifeDegenPercentTincture", mod.TypeBase, 1).Source("S"),
		), 1.237)
		testza.AssertEqual(t, game.ward, list.Sum(mod.TypeBase, nil, "WardRegenPercent"))
		testza.AssertEqual(t, game.tincture, list.Sum(mod.TypeBase, nil, "LifeDegenPercentTincture"))

		more := []mod.Mod{
			mod.NewFloat("ReservationMultiplier", mod.TypeMore, 12.345).Source("S"),
			mod.NewFloat("ReservationMultiplier", mod.TypeMore, 3.333).Source("S"),
		}
		db := dbFor(game.rules, more...)
		testza.AssertEqual(t, game.moreReservation, db.More(nil, "ReservationMultiplier"))
		ml := NewModList()
		ml.Rules = game.rules
		ml.AddMod(more[0])
		ml.AddMod(more[1])
		testza.AssertEqual(t, game.moreReservation, ml.More(nil, "ReservationMultiplier"))
	}
}

func TestRulesManaUnreservedNegative(t *testing.T) {
	for _, game := range []struct {
		rules *Rules
		want  float64
	}{{nil, 120}, {poe2HookRules(), -20}} {
		db := dbFor(game.rules, mod.NewFloat("X", mod.TypeBase, 1).Source("S").Tag(mod.PerStat(1, "ManaUnreserved")))
		db.Actor = manaActor{"ManaUnreserved": -20, "Mana": 100, "ManaUnreservedPercent": -20, "ManaEfficiency": 0}
		testza.AssertEqual(t, game.want, db.Sum(mod.TypeBase, nil, "X"))
	}
}

func TestRulesGameIDFromGemName(t *testing.T) {
	rules := poe2HookRules()
	rules.GameIDFromGemName = func(name string) string {
		return map[string]string{
			"Fireball":          "Metadata/Items/Gem/SkillGemFireball",
			"Fireball of Alpha": "Metadata/Items/Gem/SkillGemFireball",
		}[name]
	}
	db := dbFor(rules, mod.NewFloat("X", mod.TypeBase, 1).Source("S").Tag(mod.SkillName("Fireball").Transfigured(true)))
	testza.AssertEqual(t, float64(1), db.Sum(mod.TypeBase, &ListCfg{SkillName: "Fireball"}, "X"))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, &ListCfg{SkillName: "Nonexistent Skill"}, "X"))
	// Only the caller's table knows this name, so PoE1's gem table cannot be what matched it.
	testza.AssertEqual(t, float64(1), db.Sum(mod.TypeBase, &ListCfg{SkillName: "Fireball of Alpha"}, "X"))
}

func TestRulesGemIsType(t *testing.T) {
	rules := poe2HookRules()
	rules.GemIsType = func(gem any, keyword string) bool {
		tags := gem.(*poe2TestGem).tags
		return keyword == "elemental" && (tags["fire"] || tags["cold"] || tags["lightning"])
	}
	db := dbFor(rules, mod.NewFloat("X", mod.TypeBase, 1).Source("S").Tag(mod.SocketedIn("Weapon 1").Keyword("elemental")))
	fire := &ListCfg{SlotName: "Weapon 1", GemData: &poe2TestGem{tags: map[string]bool{"fire": true}}}
	chaos := &ListCfg{SlotName: "Weapon 1", GemData: &poe2TestGem{tags: map[string]bool{"chaos": true}}}
	testza.AssertEqual(t, float64(1), db.Sum(mod.TypeBase, fire, "X"))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, chaos, "X"))
}

func listOf(mods ...mod.Mod) *ModList {
	l := NewModList()
	for _, m := range mods {
		l.AddMod(m)
	}
	return l
}
