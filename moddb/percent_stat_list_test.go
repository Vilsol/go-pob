package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestPercentStatScalesAListPayload: PercentStat is the third of ModStore:EvalMod's three table-valued cases.
func TestPercentStatScalesAListPayload(t *testing.T) {
	db := NewModDB()
	cfg := &ListCfg{SkillStats: map[string]float64{"Life": 1000}}

	db.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "FireDot", Value: 3}).Tag(mod.PercentStat("Life", 20)))
	db.AddMod(mod.NewFloat("Stat", mod.TypeBase, 3).Tag(mod.PercentStat("Life", 20)))

	testza.AssertEqual(t, float64(600), db.Sum(mod.TypeBase, cfg, "Stat"))

	list := db.List(cfg, "SkillData")
	testza.AssertEqual(t, 1, len(list))
	payload, ok := list[0].(mod.SkillData)
	testza.AssertTrue(t, ok)
	testza.AssertEqual(t, float64(600), payload.Value)
	testza.AssertEqual(t, "FireDot", payload.Key)
}

// TestPercentStatListScalingDoesNotMutateTheStoredMod: The stored mod must be untouched: evalMod threads a COPY, so querying twice cannot compound the scaling.
func TestPercentStatListScalingDoesNotMutateTheStoredMod(t *testing.T) {
	db := NewModDB()
	cfg := &ListCfg{SkillStats: map[string]float64{"Life": 1000}}
	db.AddMod(mod.NewList("SkillData", mod.SkillData{Key: "FireDot", Value: 3}).Tag(mod.PercentStat("Life", 20)))

	first := db.List(cfg, "SkillData")[0].(mod.SkillData)
	second := db.List(cfg, "SkillData")[0].(mod.SkillData)
	testza.AssertEqual(t, first.Value, second.Value)
	testza.AssertEqual(t, float64(600), second.Value)
}
