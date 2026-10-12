package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

func finalizeEnv(playerCfgSource, enemyCfgSource string) *Environment {
	enemyDB := moddb.NewModDB()
	enemyDB.AddMod(mod.NewFloat("TestStat", mod.TypeBase, 7).Source("Tree"))
	enemyDB.AddMod(mod.NewFloat("TestStat", mod.TypeBase, 100).Source("Item"))

	enemy := &Actor{
		ModDB: enemyDB,
		MainSkill: &ActiveSkill{
			SkillCfg:     &moddb.ListCfg{Source: utils.Ptr(mod.Source(enemyCfgSource))},
			SkillModList: moddb.NewModList(),
		},
	}
	player := &Actor{
		ModDB: moddb.NewModDB(),
		Enemy: enemy,
		MainSkill: &ActiveSkill{
			SkillCfg: &moddb.ListCfg{Source: utils.Ptr(mod.Source(playerCfgSource))},
		},
	}
	return &Environment{Player: player, Enemy: enemy}
}

// TestFinalizeCalcTabEnemySectionReadsPlayerCfg: An enemy section takes its modStore off the enemy and its CFG off the player.
func TestFinalizeCalcTabEnemySectionReadsPlayerCfg(t *testing.T) {
	env := finalizeEnv("Tree", "Item")
	SetColProps(map[string]ColProps{
		"c": {Cfg: "skill", Enemy: utils.Ptr(true), ModName: []string{"TestStat"}, ModType: mod.TypeBase},
	})
	t.Cleanup(func() { SetColProps(nil) })

	got := FinalizeCalcTab(env)
	testza.AssertEqual(t, float64(7), got["c"],
		"cfg must come off the PLAYER's mainSkill (source Tree -> 7); 100 means it came off the enemy's (source Item)")
}

// TestFinalizeCalcTabAppliesModSourceWithoutCfg: CalcSections.lua:1444-1445 carry a modSource with no cfg at all ("Base from Gear", "Inc.
func TestFinalizeCalcTabAppliesModSourceWithoutCfg(t *testing.T) {
	env := finalizeEnv("Tree", "Item")
	SetColProps(map[string]ColProps{
		"c": {ModSource: utils.Ptr("Tree"), Enemy: utils.Ptr(true), ModName: []string{"TestStat"}, ModType: mod.TypeBase},
	})
	t.Cleanup(func() { SetColProps(nil) })

	got := FinalizeCalcTab(env)
	testza.AssertEqual(t, float64(7), got["c"],
		"modSource must filter even with no cfg; 107 is the unfiltered total, i.e. the source was dropped")
}

// TestFinalizeCalcTabSurvivesNilMainSkill: PoB guards the cfg index with `actor.mainSkill and ...` on BOTH the cfg and the modStore arm.
func TestFinalizeCalcTabSurvivesNilMainSkill(t *testing.T) {
	db := moddb.NewModDB()
	db.AddMod(mod.NewFloat("TestStat", mod.TypeBase, 5).Source("Tree"))
	env := &Environment{Player: &Actor{ModDB: db}}

	SetColProps(map[string]ColProps{
		"c": {Cfg: "skill", ModName: []string{"TestStat"}, ModType: mod.TypeBase},
	})
	t.Cleanup(func() { SetColProps(nil) })

	got := FinalizeCalcTab(env)
	testza.AssertEqual(t, float64(5), got["c"], "must fall through to actor.modDB, as the Lua disjunction does")
}

// TestFinalizeCalcTabDoesNotMutateSharedSkillCfg: PoB assigns onto the skill's own cfg table.
func TestFinalizeCalcTabDoesNotMutateSharedSkillCfg(t *testing.T) {
	env := finalizeEnv("Tree", "Item")
	SetColProps(map[string]ColProps{
		"c": {Cfg: "skill", ModSource: utils.Ptr("Item"), Enemy: utils.Ptr(true), ModName: []string{"TestStat"}, ModType: mod.TypeBase},
	})
	t.Cleanup(func() { SetColProps(nil) })

	_ = FinalizeCalcTab(env)
	src := env.Player.MainSkill.SkillCfg.Source
	testza.AssertNotNil(t, src)
	testza.AssertEqual(t, mod.Source("Tree"), *src, "the column's modSource leaked onto the shared skillCfg")
}

// TestMainSkillCfgCoversEveryStoredName: The five cfg names PoB never stores must resolve to an empty cfg.
func TestMainSkillCfgCoversEveryStoredName(t *testing.T) {
	skill := &ActiveSkill{
		SkillCfg:    &moddb.ListCfg{SlotName: "skill"},
		Weapon1Cfg:  &moddb.ListCfg{SlotName: "weapon1"},
		Weapon2Cfg:  &moddb.ListCfg{SlotName: "weapon2"},
		BleedCfg:    &moddb.ListCfg{SlotName: "bleed"},
		OHBleedCfg:  &moddb.ListCfg{SlotName: "OHbleed"},
		PoisonCfg:   &moddb.ListCfg{SlotName: "poison"},
		OHPoisonCfg: &moddb.ListCfg{SlotName: "OHpoison"},
		IgniteCfg:   &moddb.ListCfg{SlotName: "ignite"},
		OHIgniteCfg: &moddb.ListCfg{SlotName: "OHignite"},
		DecayCfg:    &moddb.ListCfg{SlotName: "decay"},
	}
	for _, name := range []string{"skill", "weapon1", "weapon2", "bleed", "OHbleed", "poison", "OHpoison", "ignite", "OHignite", "decay"} {
		cfg := mainSkillCfg(skill, name)
		testza.AssertNotNil(t, cfg, "cfg name %q is stored on the skill by PoB and must resolve", name)
		if cfg != nil {
			testza.AssertEqual(t, name, cfg.SlotName, "cfg name %q resolved to the wrong field", name)
		}
	}
	for _, name := range []string{"dotPhysical", "dotLightning", "dotCold", "dotFire", "dotChaos"} {
		testza.AssertNil(t, mainSkillCfg(skill, name), "cfg name %q is not stored by PoB and must not invent a cfg", name)
	}
}

// TestFinalizeCalcTabShowsTheMinion: with Show Minion Stats on, a CALCS pass's sections read the minion.
func TestFinalizeCalcTabShowsTheMinion(t *testing.T) {
	playerDB := moddb.NewModDB()
	playerDB.AddMod(mod.NewFloat("TestStat", mod.TypeBase, 5).Source("Tree"))
	minionDB := moddb.NewModDB()
	minionDB.AddMod(mod.NewFloat("TestStat", mod.TypeBase, 3).Source("Tree"))
	build := &pob.PathOfBuilding{}
	env := &Environment{Build: build, Mode: OutputModeCalcs, Player: &Actor{ModDB: playerDB}, Minion: &ActorMinion{Actor: &Actor{ModDB: minionDB}}}
	SetColProps(map[string]ColProps{"c": {ModName: []string{"TestStat"}, ModType: mod.TypeBase}})
	t.Cleanup(func() { SetColProps(nil) })

	testza.AssertEqual(t, float64(5), FinalizeCalcTab(env)["c"], "Show Minion Stats off: the player's")
	build.SetCalcsInputBool("showMinion", true)
	testza.AssertEqual(t, float64(3), FinalizeCalcTab(env)["c"], "Show Minion Stats on: the minion's")
	env.Mode = OutputModeMain
	testza.AssertEqual(t, float64(5), FinalizeCalcTab(env)["c"], "only the Calcs tab's pass shows the minion")
}
