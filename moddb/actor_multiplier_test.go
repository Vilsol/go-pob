package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

type outputActor struct {
	modDB   *ModDB
	links   map[string]*outputActor
	outputs map[string]float64
}

func (a *outputActor) GetOutput(stat string) (float64, bool) {
	v, ok := a.outputs[stat]
	return v, ok
}

func (a *outputActor) RelatedModDB(name string) ModStoreFuncs {
	related, ok := a.links[name]
	if !ok || related == nil || related.modDB == nil {
		return nil
	}
	return related.modDB
}

func minionAndParent() (*ModDB, *outputActor) {
	minionDB := NewModDB()
	parentDB := NewModDB()

	parent := &outputActor{modDB: parentDB, links: map[string]*outputActor{}, outputs: map[string]float64{}}
	minion := &outputActor{modDB: minionDB, links: map[string]*outputActor{"parent": parent}, outputs: map[string]float64{}}

	parentDB.Actor = parent
	minionDB.Actor = minion

	return minionDB, parent
}

// TestMultiplierUsesNamedActorsMultiplier pins ModStore.lua:262-268.
func TestMultiplierUsesNamedActorsMultiplier(t *testing.T) {
	playerDB, enemyDB := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, 6).Tag(mod.Multiplier("ImpaleStacks").Actor("enemy")))

	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeBase, nil, "PhysicalMin"))

	enemyDB.Multipliers["ImpaleStacks"] = 5
	testza.AssertEqual(t, 30.0, playerDB.Sum(mod.TypeBase, nil, "PhysicalMin"))
}

// TestMultiplierIgnoresOwnActorsMultiplier is the mirror case: a multiplier of the same name on the store the mod lives on must not satisfy an actor-scoped tag.
func TestMultiplierIgnoresOwnActorsMultiplier(t *testing.T) {
	playerDB, _ := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, 6).Tag(mod.Multiplier("ImpaleStacks").Actor("enemy")))

	playerDB.Multipliers["ImpaleStacks"] = 5
	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeBase, nil, "PhysicalMin"))
}

// TestMultiplierMissingActorExcludesMod ports the `else return end` at ModStore.lua:265-267.
func TestMultiplierMissingActorExcludesMod(t *testing.T) {
	db := NewModDB()
	db.Multipliers["ImpaleStacks"] = 5
	db.AddMod(mod.NewFloat("PhysicalMin", mod.TypeBase, 6).Tag(mod.Multiplier("ImpaleStacks").Actor("enemy").Base(100)))

	testza.AssertEqual(t, 0.0, db.Sum(mod.TypeBase, nil, "PhysicalMin"))
}

// TestMultiplierLimitActorUsesNamedActor ports ModStore.lua:255-261.
func TestMultiplierLimitActorUsesNamedActor(t *testing.T) {
	minionDB, parent := minionAndParent()
	parent.modDB.Multipliers["VirulenceStack"] = 30
	parent.modDB.Multipliers["VirulenceStacksMax"] = 20

	minionDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 1).
		Tag(mod.Multiplier("VirulenceStack").Actor("parent").LimitVar("VirulenceStacksMax").LimitActor("parent")))

	testza.AssertEqual(t, 20.0, minionDB.Sum(mod.TypeIncrease, nil, "PhysicalDamage"))
}

// TestMultiplierLimitActorMissingExcludesMod ports the `else return end` at :258-260.
func TestMultiplierLimitActorMissingExcludesMod(t *testing.T) {
	db := NewModDB()
	db.Multipliers["VirulenceStack"] = 3
	db.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 1).
		Tag(mod.Multiplier("VirulenceStack").LimitVar("VirulenceStacksMax").LimitActor("parent")))

	testza.AssertEqual(t, 0.0, db.Sum(mod.TypeIncrease, nil, "PhysicalDamage"))
}

// TestMultiplierThresholdUsesNamedActor pins ModStore.lua:308-314.
func TestMultiplierThresholdUsesNamedActor(t *testing.T) {
	playerDB, enemyDB := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).
		Tag(mod.MultiplierThreshold("PoisonStack").Actor("enemy").Threshold(3)))

	playerDB.Multipliers["PoisonStack"] = 5
	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeIncrease, nil, "Damage"))

	enemyDB.Multipliers["PoisonStack"] = 5
	testza.AssertEqual(t, 40.0, playerDB.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestMultiplierThresholdMissingActorExcludesMod ports :311-313's `else return end`.
func TestMultiplierThresholdMissingActorExcludesMod(t *testing.T) {
	db := NewModDB()
	db.Multipliers["PoisonStack"] = 5
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).
		Tag(mod.MultiplierThreshold("PoisonStack").Actor("enemy").Threshold(3)))

	testza.AssertEqual(t, 0.0, db.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestPerStatUsesNamedActorsStat pins ModStore.lua:332-334.
func TestPerStatUsesNamedActorsStat(t *testing.T) {
	minionDB, parent := minionAndParent()
	parent.outputs["Life"] = 1000

	minionDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.PerStat(100, "Life").Actor("parent")))

	testza.AssertEqual(t, 10.0, minionDB.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestPerStatMissingActorFallsBackToSelf is the case that differs from Multiplier.
func TestPerStatMissingActorFallsBackToSelf(t *testing.T) {
	db := NewModDB()
	db.Actor = &outputActor{outputs: map[string]float64{"Life": 500}}
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 1).Tag(mod.PerStat(100, "Life").Actor("parent")))

	testza.AssertEqual(t, 5.0, db.Sum(mod.TypeIncrease, nil, "Damage"))
}
