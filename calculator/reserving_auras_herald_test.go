package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/data"
)

// TestAuraReservedBaseFindsHeraldsNotJustAuras: ModStore.lua:284 gates the reservation walk on `SkillType.HasReservation`, NOT on `SkillType.Aura`.
func TestAuraReservedBaseFindsHeraldsNotJustAuras(t *testing.T) {
	herald := &ActiveSkill{
		SkillTypes: map[data.SkillType]bool{
			data.SkillTypeHasReservation: true,
			data.SkillTypeHerald:         true,
			data.SkillTypeBuff:           true,
		},
		SkillFlags: map[SkillFlag]bool{},
		SkillData:  &SkillData{ManaReservedBase: 132, LifeReservedBase: 44},
		BuffList:   []*Buff{{Name: "Herald of Ice"}},
	}
	actor := &Actor{ActiveSkillList: []*ActiveSkill{herald}}

	base, found := actor.AuraManaReservedBase("Herald of Ice", "")
	testza.AssertTrue(t, found, "a HasReservation skill must be found even without SkillType.Aura")
	testza.AssertEqual(t, float64(132), base)
}
