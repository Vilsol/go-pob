package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type gateActor struct{}

func (gateActor) GetOutput(string) (float64, bool) { return 0, false }

type gateGrantedEffect struct{ id string }

func (g gateGrantedEffect) ID() string { return g.id }

func TestEvalModGatingTags(t *testing.T) {
	tests := []struct {
		name       string
		tag        mod.Tag
		matchCfg   *ListCfg
		nomatchCfg *ListCfg
	}{
		{
			name:       "SkillType",
			tag:        mod.SkillType("Attack"),
			matchCfg:   &ListCfg{SkillTypes: map[data.SkillType]bool{data.SkillTypeAttack: true}},
			nomatchCfg: &ListCfg{SkillTypes: map[data.SkillType]bool{data.SkillTypeSpell: true}},
		},
		{
			name:       "SkillType-neg",
			tag:        mod.SkillType("Attack").Neg(true),
			matchCfg:   &ListCfg{SkillTypes: map[data.SkillType]bool{data.SkillTypeSpell: true}},
			nomatchCfg: &ListCfg{SkillTypes: map[data.SkillType]bool{data.SkillTypeAttack: true}},
		},
		{
			name:       "SkillName",
			tag:        mod.SkillName("Fireball"),
			matchCfg:   &ListCfg{SkillName: "Fireball"},
			nomatchCfg: &ListCfg{SkillName: "Icebolt"},
		},
		{
			name:       "SkillName-neg",
			tag:        mod.SkillName("Fireball").Neg(true),
			matchCfg:   &ListCfg{SkillName: "Icebolt"},
			nomatchCfg: &ListCfg{SkillName: "Fireball"},
		},
		{
			name:       "SkillId",
			tag:        mod.SkillId("abc"),
			matchCfg:   &ListCfg{SkillGrantedEffect: gateGrantedEffect{id: "abc"}},
			nomatchCfg: &ListCfg{SkillGrantedEffect: gateGrantedEffect{id: "xyz"}},
		},
		{
			name:       "SlotName",
			tag:        mod.SlotName("Weapon 1"),
			matchCfg:   &ListCfg{SlotName: "Weapon 1"},
			nomatchCfg: &ListCfg{SlotName: "Body Armour"},
		},
		{
			name:       "ModFlagOr",
			tag:        mod.ModFlagOr(mod.MFlagAttack),
			matchCfg:   &ListCfg{Flags: utils.Ptr(mod.MFlagAttack)},
			nomatchCfg: &ListCfg{Flags: utils.Ptr(mod.MFlagProjectile)},
		},
		{
			name:       "StatThreshold-lower",
			tag:        mod.StatThreshold("Life", 100),
			matchCfg:   &ListCfg{SkillStats: map[string]float64{"Life": 150}},
			nomatchCfg: &ListCfg{SkillStats: map[string]float64{"Life": 50}},
		},
		{
			name:       "StatThreshold-upper",
			tag:        mod.StatThreshold("Life", 100).Upper(true),
			matchCfg:   &ListCfg{SkillStats: map[string]float64{"Life": 50}},
			nomatchCfg: &ListCfg{SkillStats: map[string]float64{"Life": 150}},
		},
		{
			name:       "StatThreshold-thresholdPercent",
			tag:        mod.StatThresholdStat("Life", "Mana").Percent(25),
			matchCfg:   &ListCfg{SkillStats: map[string]float64{"Life": 30, "Mana": 100}},
			nomatchCfg: &ListCfg{SkillStats: map[string]float64{"Life": 20, "Mana": 100}},
		},
		{
			name:       "StatThreshold-thresholdPercent-boundary",
			tag:        mod.StatThresholdStat("Life", "Mana").Percent(25),
			matchCfg:   &ListCfg{SkillStats: map[string]float64{"Life": 25, "Mana": 100}},
			nomatchCfg: &ListCfg{SkillStats: map[string]float64{"Life": 24.9, "Mana": 100}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := NewModDB()
			db.Actor = gateActor{}
			db.AddMod(mod.NewFloat("test", mod.TypeBase, 5).Tag(test.tag))
			testza.AssertEqual(t, 5.0, db.Sum(mod.TypeBase, test.matchCfg, "test"), "match cfg should include the mod")
			testza.AssertEqual(t, 0.0, db.Sum(mod.TypeBase, test.nomatchCfg, "test"), "non-match cfg should gate the mod")
		})
	}
}
