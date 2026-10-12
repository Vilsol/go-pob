package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

type reservingActor struct {
	output map[string]float64
	auras  map[string]float64
}

func (a *reservingActor) GetOutput(key string) (float64, bool) {
	v, ok := a.output[key]
	return v, ok
}

func (a *reservingActor) AuraManaReservedBase(skillName, summonSkillName string) (float64, bool) {
	if v, ok := a.auras[skillName]; ok {
		return v, true
	}
	v, ok := a.auras[summonSkillName]
	return v, ok
}

func (a *reservingActor) AuraLifeReservedBase(string, string) (float64, bool) { return 0, false }

// TestGetStatManaReservedPercentIsPerSkill pins ModStore.lua:222-232, which is a SPECIAL CASE that must beat the output lookup rather than fall through to it.
func TestGetStatManaReservedPercentIsPerSkill(t *testing.T) {
	actor := &reservingActor{
		output: map[string]float64{"Mana": 1000, "ManaReservedPercent": 98},
		auras:  map[string]float64{"Purity of Fire": 250},
	}
	store := NewModDB()
	store.Actor = actor

	testza.AssertEqual(t, float64(25), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Purity of Fire"}))

	testza.AssertEqual(t, float64(0), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Cyclone"}))

	testza.AssertEqual(t, float64(0), GetStat(store, "ManaReservedPercent", nil))
}

// TestGetStatManaReservedPercentClampsAtOneHundred is the `m_min(..., 100)` on ModStore.lua:231.
func TestGetStatManaReservedPercentClampsAtOneHundred(t *testing.T) {
	actor := &reservingActor{
		output: map[string]float64{"Mana": 100},
		auras:  map[string]float64{"Grace": 500},
	}
	store := NewModDB()
	store.Actor = actor

	testza.AssertEqual(t, float64(100), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Grace"}))
}

// TestGetStatManaReservedPercentWithoutTheSeam covers the actor that does NOT implement ReservingAuras — a standalone ModList, or the enemy.
func TestGetStatManaReservedPercentWithoutTheSeam(t *testing.T) {
	store := NewModDB()
	store.Actor = &plainActor{output: map[string]float64{"Mana": 1000, "ManaReservedPercent": 98}}

	testza.AssertEqual(t, float64(0), GetStat(store, "ManaReservedPercent", &ListCfg{SkillName: "Grace"}))

	testza.AssertEqual(t, float64(1000), GetStat(store, "Mana", nil))
}

type plainActor struct{ output map[string]float64 }

func (a *plainActor) GetOutput(key string) (float64, bool) {
	v, ok := a.output[key]
	return v, ok
}

// TestPerStatTagReadsThePerSkillReservation is the end-to-end shape mod_parser.go:1222 produces.
func TestPerStatTagReadsThePerSkillReservation(t *testing.T) {
	store := NewModDB()
	store.Actor = &reservingActor{
		output: map[string]float64{"Mana": 1000, "ManaReservedPercent": 98},
		auras:  map[string]float64{"Purity of Fire": 500},
	}
	store.AddMod(mod.NewFloat("AuraEffect", mod.TypeMore, 1).Tag(mod.PerStat(2, "ManaReservedPercent")))

	testza.AssertEqual(t, 1.25, store.More(&ListCfg{SkillName: "Purity of Fire"}, "AuraEffect"))

	testza.AssertEqual(t, float64(1), store.More(&ListCfg{SkillName: "Cyclone"}, "AuraEffect"))
}
