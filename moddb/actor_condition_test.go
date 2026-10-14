package moddb

import (
	"sort"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

type linkedActor struct {
	modDB *ModDB
	links map[string]*linkedActor

	equipped map[string]fakeEquippedItem

	monsterTags []string
}

func (a *linkedActor) MonsterTags() ([]string, bool) { return a.monsterTags, a.monsterTags != nil }

type fakeEquippedItem struct {
	modifiers []string
	rarity    string
	corrupted bool
	name      string
	shaper    bool
	elder     bool
}

func (a *linkedActor) ItemShaperIs(slot string, shaper bool) bool {
	item, ok := a.equipped[slot]
	return ok && item.shaper == shaper
}

func (a *linkedActor) ItemElderIs(slot string, elder bool) bool {
	item, ok := a.equipped[slot]
	return ok && item.elder == elder
}

func (a *linkedActor) GetOutput(string) (float64, bool) { return 0, false }

func (a *linkedActor) EquippedSlots() []string {
	slots := make([]string, 0, len(a.equipped))
	for slot := range a.equipped {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	return slots
}

func (a *linkedActor) ItemInSlot(slot string) bool {
	_, ok := a.equipped[slot]
	return ok
}

func (a *linkedActor) ItemModifierMatches(slot, searchCond string) bool {
	for _, m := range a.equipped[slot].modifiers {
		if strings.Contains(strings.ToLower(m), searchCond) {
			return true
		}
	}
	return false
}

func (a *linkedActor) ItemRarityIs(slot, rarity string) bool {
	return strings.EqualFold(a.equipped[slot].rarity, rarity)
}

func (a *linkedActor) ItemCorruptedIs(slot string, corrupted bool) bool {
	return a.equipped[slot].corrupted == corrupted
}

func (a *linkedActor) ItemNameIs(slot, name string) bool {
	return strings.EqualFold(a.equipped[slot].name, name)
}

func (a *linkedActor) RelatedModDB(name string) ModStoreFuncs {
	related, ok := a.links[name]
	if !ok || related == nil || related.modDB == nil {
		return nil
	}
	return related.modDB
}

func playerAndEnemy() (*ModDB, *ModDB) {
	playerDB := NewModDB()
	enemyDB := NewModDB()

	player := &linkedActor{modDB: playerDB, links: map[string]*linkedActor{}}
	enemy := &linkedActor{modDB: enemyDB, links: map[string]*linkedActor{}}
	player.links["enemy"] = enemy
	enemy.links["enemy"] = player

	playerDB.Actor = player
	enemyDB.Actor = enemy

	return playerDB, enemyDB
}

// TestActorConditionUsesNamedActorsConditions pins ModStore.lua:478-501.
func TestActorConditionUsesNamedActorsConditions(t *testing.T) {
	playerDB, enemyDB := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.ActorCondition("enemy", "Shocked")))

	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeIncrease, nil, "Damage"))

	enemyDB.Conditions["Shocked"] = true
	testza.AssertEqual(t, 40.0, playerDB.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestActorConditionIgnoresOwnActorsConditions is the mirror case: the PLAYER carrying the condition must NOT satisfy an enemy-scoped tag.
func TestActorConditionIgnoresOwnActorsConditions(t *testing.T) {
	playerDB, _ := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.ActorCondition("enemy", "Shocked")))

	playerDB.Conditions["Shocked"] = true
	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestActorConditionOnEnemyStoreResolvesToPlayer covers the other direction of the mutual link.
func TestActorConditionOnEnemyStoreResolvesToPlayer(t *testing.T) {
	playerDB, enemyDB := playerAndEnemy()
	enemyDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Tag(mod.ActorCondition("enemy", "Fortified")))

	enemyDB.Conditions["Fortified"] = true
	testza.AssertEqual(t, 0.0, enemyDB.Sum(mod.TypeIncrease, nil, "DamageTaken"))

	playerDB.Conditions["Fortified"] = true
	testza.AssertEqual(t, 10.0, enemyDB.Sum(mod.TypeIncrease, nil, "DamageTaken"))
}

// TestActorConditionMissingLinkIsNotAnError ports the `target = self.actor[tag.actor] and self.actor[tag.actor].modDB` / `if target then` pair.
func TestActorConditionMissingLinkIsNotAnError(t *testing.T) {
	db := NewModDB()
	db.Conditions["Shocked"] = true
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.ActorCondition("enemy", "Shocked")))
	db.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 5).Tag(mod.ActorCondition("enemy", "Shocked").Neg(true)))

	testza.AssertEqual(t, 0.0, db.Sum(mod.TypeIncrease, nil, "Damage"))
	testza.AssertEqual(t, 5.0, db.Sum(mod.TypeIncrease, nil, "Speed"))
}

// TestActorConditionInheritsActorThroughParent pins that the link is resolved through the store's inherited actor.
func TestActorConditionInheritsActorThroughParent(t *testing.T) {
	playerDB, enemyDB := playerAndEnemy()

	list := NewModList()
	list.Parent = playerDB
	list.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.ActorCondition("enemy", "Shocked")))

	testza.AssertEqual(t, 0.0, list.Sum(mod.TypeIncrease, nil, "Damage"))

	enemyDB.Conditions["Shocked"] = true
	testza.AssertEqual(t, 40.0, list.Sum(mod.TypeIncrease, nil, "Damage"))
}

// TestActorConditionDoesNotReadSkillCond pins the difference between PoB's Condition case.
func TestActorConditionDoesNotReadSkillCond(t *testing.T) {
	playerDB, _ := playerAndEnemy()
	playerDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 40).Tag(mod.ActorCondition("enemy", "Shocked")))

	cfg := &ListCfg{SkillCond: map[string]bool{"Shocked": true}}
	testza.AssertEqual(t, 0.0, playerDB.Sum(mod.TypeIncrease, cfg, "Damage"))
}
