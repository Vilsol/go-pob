package calculator2

import (
	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob2"
)

// Environment is CalcSetup's env: the build's actors, databases and allocated tree.
//
//pob2:block:src/Modules/CalcSetup.lua:857-914
type Environment struct {
	Build *pob2.PathOfBuilding2
	Mode  string

	Spec    *tree2.PassiveSpec
	ClassID int

	ModDB     *moddb.ModDB
	EnemyDB   *moddb.ModDB
	ItemModDB *moddb.ModDB

	EnemyLevel int
	Player     *Actor
	Enemy      *Actor

	// ConfigInput is build.configTab.input for the active config set.
	ConfigInput map[string]ConfigValue

	ModeBuffs     bool
	ModeCombat    bool
	ModeEffective bool

	MainSocketGroup int
	WeaponSet       int

	AllocNodes      map[int]*tree2.SpecNode
	GrantedPassives map[int]bool
	KeystonesAdded  map[string]bool

	// nodeModLists holds each tree node's ProcessStats modList, keyed by node id.
	nodeModLists map[int]*moddb.ModList
	modSource    ModSource
	items        Items
}

// Actor is env.player / env.enemy.
type Actor struct {
	ModDB  *moddb.ModDB
	Level  int
	Output map[string]float64

	Player *Actor
	Enemy  *Actor
	Parent *Actor

	// ItemList is actor.itemList: the equipped item per slot name.
	ItemList map[string]*Item
	// WeaponData1Type and WeaponData2Type are actor.weaponData1/2.type; "" is nil.
	WeaponData1Type string
	WeaponData2Type string
}

func (a *Actor) GetOutput(key string) (float64, bool) {
	if a == nil || a.Output == nil {
		return 0, false
	}
	v, ok := a.Output[key]
	return v, ok
}

//pob2:src/Classes/ModStore.lua:73-79
func (a *Actor) RelatedModDB(name string) moddb.ModStoreFuncs {
	var other *Actor
	switch name {
	case "player":
		switch {
		case a.Player != nil:
			other = a.Player
		case a.Parent != nil && a.Parent.Player != nil:
			other = a.Parent.Player
		case a.Enemy != nil:
			other = a.Enemy.Player
		}
	case "enemy":
		other = a.Enemy
	case "parent":
		other = a.Parent
	}
	if other == nil || other.ModDB == nil {
		return nil
	}
	return other.ModDB
}

// ConfigValue is one config input: a number, a string or a boolean, as the XML stored it.
type ConfigValue struct {
	Number    float64
	String    string
	Boolean   bool
	IsNumber  bool
	IsString  bool
	IsBoolean bool
}
