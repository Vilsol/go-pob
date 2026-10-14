package moddb

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

type fakeGrantedEffect string

func (f fakeGrantedEffect) ID() string { return string(f) }

var evalTagCaseRe = regexp.MustCompile(`(?m)^\tcase \*mod\.(\w+):`)

func traceCoverageCases(t *testing.T) (map[string]map[string]mod.Mod, *ModDB, *ListCfg) {
	t.Helper()

	playerDB, enemyDB := playerAndEnemy()
	enemyDB.Conditions["Cursed"] = true
	playerDB.Multipliers["A"] = 3
	playerDB.Conditions["Met"] = true

	playerDB.Actor.(*linkedActor).equipped = map[string]fakeEquippedItem{
		"Body Armour": {modifiers: []string{"+50 to maximum Life"}, rarity: "RARE"},
	}
	playerDB.Actor.(*linkedActor).monsterTags = []string{"construct"}

	attack := mod.MFlagAttack
	arrowAttack := mod.KeywordFlagArrow | mod.KeywordFlagAttack
	cfg := &ListCfg{
		Flags:              &attack,
		KeywordFlags:       &arrowAttack,
		SkillStats:         map[string]float64{"Strength": 200, "Life": 1000},
		SkillName:          "Cyclone",
		SummonSkillName:    "Raise Spectre",
		SkillPart:          2,
		SlotName:           "Body Armour",
		SkillTypes:         map[data.SkillType]bool{"Melee": true},
		SkillGrantedEffect: fakeGrantedEffect("Cyclone"),
	}

	base := func() mod.Mod { return mod.NewFloat("Stat", mod.TypeBase, 100) }

	return map[string]map[string]mod.Mod{
		"ItemConditionTag": {
			"admits":      base().Tag(mod.ItemCondition("Body Armour").Search("life")),
			"gates":       base().Tag(mod.ItemCondition("Body Armour").Search("mana")),
			"admitsEmpty": base().Tag(mod.ItemCondition("Helmet").Search("life").Negate(true)),
			"gatesEmpty":  base().Tag(mod.ItemCondition("Helmet").Search("life")),
		},
		"MultiplierTag": {
			"admits": base().Tag(mod.Multiplier("A")),
			"gates":  base().Tag(mod.Multiplier("A").Actor("nobody")),
		},
		"MultiplierThresholdTag": {
			"admits": base().Tag(mod.MultiplierThreshold("A").Threshold(2)),
			"gates":  base().Tag(mod.MultiplierThreshold("A").Threshold(9)),
		},
		"PerStatTag": {
			"admits": base().Tag(mod.PerStat(100, "Strength")),
		},
		"PercentStatTag": {
			"admits": base().Tag(mod.PercentStat("Strength", 50)),
		},
		"DistanceRampTag": {
			"gates": base().Tag(mod.DistanceRamp([][]float64{{0, 1}, {10, 0.5}})),
		},
		"MeleeProximityTag": {
			"gates": base().Tag(mod.MeleeProximity([]int{1, 0})),
		},
		"LimitTag": {
			"admits": base().Tag(mod.Limit(40)),
		},
		"ConditionTag": {
			"admits": base().Tag(mod.Condition("Met")),
			"gates":  base().Tag(mod.Condition("Unmet")),
		},
		"ActorConditionTag": {
			"admits": base().Tag(mod.ActorCondition("enemy", "Cursed")),
			"gates":  base().Tag(mod.ActorCondition("enemy", "Shocked")),
		},
		"SkillNameTag": {
			"admits": base().Tag(mod.SkillName("Cyclone")),
			"gates":  base().Tag(mod.SkillName("Raise Spectre")),
			"summon": base().Tag(mod.SkillName("Raise Spectre").SummonSkill(true)),
		},
		"SkillIDTag": {
			"admits": base().Tag(mod.SkillId("Cyclone")),
			"gates":  base().Tag(mod.SkillId("Bladestorm")),
		},
		"SkillPartTag": {
			"admits": base().Tag(mod.SkillPart(2)),
			"gates":  base().Tag(mod.SkillPart(3)),
		},
		"SkillTypeTag": {
			"admits": base().Tag(mod.SkillType("Melee")),
			"gates":  base().Tag(mod.SkillType("Spell")),
		},
		"SlotNameTag": {
			"admits": base().Tag(mod.SlotName("Body Armour")),
			"gates":  base().Tag(mod.SlotName("Helmet")),
		},
		"SocketedInTag": {
			"admits": base().Tag(mod.SocketedIn("Body Armour")),
			"gates":  base().Tag(mod.SocketedIn("Helmet")),
		},
		"ModFlagOrTag": {
			"admits": base().Tag(mod.ModFlagOr(mod.MFlagAttack)),
			"gates":  base().Tag(mod.ModFlagOr(mod.MFlagSpell)),
		},
		"StatThresholdTag": {
			"admits": base().Tag(mod.StatThreshold("Strength", 100)),
			"gates":  base().Tag(mod.StatThreshold("Strength", 900)),
		},
		"KeywordFlagAndTag": {
			"admits": base().Tag(mod.KeywordFlagAnd(mod.KeywordFlagArrow)),
			"gates":  base().Tag(mod.KeywordFlagAnd(mod.KeywordFlagArrow | mod.KeywordFlagSpell)),
		},
		"MonsterTagTag": {
			"admits": base().Tag(mod.MonsterTag("Construct")),
			"gates":  base().Tag(mod.MonsterTag("Skeleton")),
		},
	}, playerDB, cfg
}

// TestTraceEvalModCoversEveryDispatchedTag is the coverage half: every `case *mod.XTag:` in evalTag must have a pin below.
func TestTraceEvalModCoversEveryDispatchedTag(t *testing.T) {
	src, err := os.ReadFile("mod_store.go")
	if err != nil {
		t.Fatal(err)
	}
	found := evalTagCaseRe.FindAllStringSubmatch(string(src), -1)
	if len(found) < 10 {
		t.Fatalf("read only %d `case *mod.XTag:` lines out of evalTag's switch — expected the "+
			"whole dispatch; this test would pass vacuously", len(found))
	}

	cases, _, _ := traceCoverageCases(t)
	var missing []string
	for _, m := range found {
		if _, ok := cases[m[1]]; !ok {
			missing = append(missing, m[1])
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("evalTag dispatches %v with no tracer pin. eval_trace.go is the only instrument "+
			"that shows what a tag EVALUATED to, and an unpinned tag type is one nobody has "+
			"confirmed it reports at all. Add a case to traceCoverageCases.", missing)
	}
}

// TestTraceCoverageShapesAreTwoSided is the NON-VACUITY guard.
func TestTraceCoverageShapesAreTwoSided(t *testing.T) {
	cases, db, cfg := traceCoverageCases(t)

	for tagType, shapes := range cases {
		var admits, gates int
		for shape, m := range shapes {
			got := db.evalMod(m, cfg)
			switch {
			case strings.HasPrefix(shape, "admits"):
				admits++
				if got == nil {
					t.Errorf("%s/%s is named admits but the engine GATED it — this pin is vacuous",
						tagType, shape)
				}
			case strings.HasPrefix(shape, "gates"):
				gates++
				if got != nil {
					t.Errorf("%s/%s is named gates but the engine ADMITTED it (%v)", tagType, shape, got)
				}
			}
		}
		if admits == 0 && gates == 0 {
			t.Errorf("%s has no admits*/gates* shape, so nothing pins its direction", tagType)
		}
	}
}

// TestTraceEvalModAgreesOnEveryTagType is the agreement half.
func TestTraceEvalModAgreesOnEveryTagType(t *testing.T) {
	cases, db, cfg := traceCoverageCases(t)

	for tagType, shapes := range cases {
		for shape, m := range shapes {
			t.Run(tagType+"/"+shape, func(t *testing.T) {
				want := db.evalMod(m, cfg)
				got := TraceEvalMod(db, m, cfg)

				testza.AssertEqual(t, want == nil, got.Excluded())
				if want == nil {
					testza.AssertEqual(t, 1, len(got.Steps))
					testza.AssertTrue(t, got.Steps[0].After == nil)
					return
				}
				testza.AssertEqual(t, want.Type(), got.Final.Type())
				testza.AssertEqual(t, want.Float(), got.Final.Float())
				testza.AssertEqual(t, len(m.Tags()), len(got.Steps))
				testza.AssertEqual(t, float64(100), m.Value().Float())
			})
		}
	}
}
