package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestSkillNameSummonSkillArm: A SkillName tag carrying summonSkill matches cfg.summonSkillName, NOT cfg.skillName (ModStore.lua:508).
func TestSkillNameSummonSkillArm(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Stat", mod.TypeBase, 10).Tag(mod.SkillName("Raise Spectre").SummonSkill(true)))
	db.AddMod(mod.NewFloat("Other", mod.TypeBase, 10).Tag(mod.SkillName("Raise Spectre")))

	granted := &ListCfg{SkillName: "Flesh Offering", SummonSkillName: "Raise Spectre"}
	testza.AssertEqual(t, float64(10), db.Sum(mod.TypeBase, granted, "Stat"))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, granted, "Other"))

	summoner := &ListCfg{SkillName: "Raise Spectre"}
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, summoner, "Stat"))
	testza.AssertEqual(t, float64(10), db.Sum(mod.TypeBase, summoner, "Other"))

	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeBase, nil, "Stat"))
}
