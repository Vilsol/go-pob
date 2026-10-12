package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

type lifeReservingActor struct {
	output map[string]float64
	mana   map[string]float64
	life   map[string]float64
}

func (a *lifeReservingActor) GetOutput(key string) (float64, bool) {
	v, ok := a.output[key]
	return v, ok
}

func (a *lifeReservingActor) AuraManaReservedBase(skillName, summonSkillName string) (float64, bool) {
	if v, ok := a.mana[skillName]; ok {
		return v, true
	}
	v, ok := a.mana[summonSkillName]
	return v, ok
}

func (a *lifeReservingActor) AuraLifeReservedBase(skillName, summonSkillName string) (float64, bool) {
	if v, ok := a.life[skillName]; ok {
		return v, true
	}
	v, ok := a.life[summonSkillName]
	return v, ok
}

// TestGetStatReservedPercentFloors pins the `m_floor` on ModStore.lua:286 and :300.
func TestGetStatReservedPercentFloors(t *testing.T) {
	actor := &lifeReservingActor{
		output: map[string]float64{"Mana": 1000, "Life": 800, "ManaReservedPercent": 98, "LifeReservedPercent": 97},
		mana:   map[string]float64{"Herald of Ice": 132},
		life:   map[string]float64{"Herald of Thunder": 132},
	}
	store := NewModDB()
	store.Actor = actor

	testza.AssertEqual(t, float64(13), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Herald of Ice"}))
	testza.AssertEqual(t, float64(16), GetStat(store, "LifeReservedPercent", &ListCfg{SkillName: "Herald of Thunder"}))
}

// TestGetStatLifeReservedPercentIsPerSkill is the mana block's assertion for life.
func TestGetStatLifeReservedPercentIsPerSkill(t *testing.T) {
	actor := &lifeReservingActor{
		output: map[string]float64{"Mana": 1000, "Life": 1000, "LifeReservedPercent": 97},
		life:   map[string]float64{"Herald of Purity": 250},
	}
	store := NewModDB()
	store.Actor = actor

	testza.AssertEqual(t, float64(25), GetStat(store, "LifeReservedPercent", &ListCfg{SkillName: "Herald of Purity"}))
	testza.AssertEqual(t, float64(0), GetStat(store, "LifeReservedPercent", &ListCfg{SkillName: "Cyclone"}))
	testza.AssertEqual(t, float64(0), GetStat(store, "LifeReservedPercent", nil))
}

// TestGetStatReservedPercentMatchesSummonSkillName covers the `or isNameInBuffList(buffList, cfg.summonSkillName)` arm of ModStore.lua:284.
func TestGetStatReservedPercentMatchesSummonSkillName(t *testing.T) {
	actor := &lifeReservingActor{
		output: map[string]float64{"Mana": 1000},
		mana:   map[string]float64{"Herald of Agony": 250},
	}
	store := NewModDB()
	store.Actor = actor

	testza.AssertEqual(t, float64(25), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Agony Crawler", SummonSkillName: "Herald of Agony"}))
}
