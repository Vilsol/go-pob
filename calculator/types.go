package calculator

import (
	"go-pob/data"
	"go-pob/pob"
)

type Calculator struct {
	PoB *pob.PathOfBuilding
}

type OutputMode string

const (
	OutputModeMain  = OutputMode("MAIN")
	OutputModeCalcs = OutputMode("CALCS")
)

type BuffMode string

const (
	BuffModeEffective = BuffMode("EFFECTIVE")
	BuffModeCombat    = BuffMode("COMBAT")
	BuffModeBuffed    = BuffMode("BUFFED")
	BuffModeUnbuffed  = BuffMode("UNBUFFED")
)

type Environment struct {
	Build *pob.PathOfBuilding
	Mode  OutputMode
	Spec  *PassiveSpec

	ModDB      *ModDB
	EnemyModDB *ModDB
	ItemModDB  *ModDB
	Minion     *ModDB

	EnemyLevel int

	Player *Actor
	Enemy  *Actor

	RequirementsTableItems map[string]interface{} // TODO Implement
	RequirementsTableGems  map[string]interface{} // TODO Implement

	RadiusJewelList     map[string]interface{} // TODO Implement
	ExtraRadiusNodeList map[string]interface{} // TODO Implement
	GrantedSkills       map[string]interface{} // TODO Implement
	GrantedSkillsNodes  map[string]interface{} // TODO Implement
	GrantedSkillsItems  map[string]interface{} // TODO Implement
	Flasks              map[string]interface{} // TODO Implement

	GrantedPassives map[string]interface{} // TODO Implement

	AuxSkillList map[string]interface{} // TODO Implement

	ModeBuffs     bool
	ModeCombat    bool
	ModeEffective bool

	KeystonesAdded  map[string]interface{}
	MainSocketGroup int
}

type Actor struct {
	ModDB           *ModDB
	Level           int
	Enemy           *Actor                 `json:"-"`
	ItemList        map[string]interface{} // TODO Implement
	ActiveSkillList []*ActiveSkill
	Output          map[string]float64
	OutputTable     map[OutTable]map[string]float64
	MainSkill       *ActiveSkill           // TODO Implement
	Breakdown       interface{}            // TODO Implement
	WeaponData1     map[string]interface{} // TODO Implement. Might be SomeSource?
	WeaponData2     map[string]interface{} // TODO Implement. Might be SomeSource?
	StrDmgBonus     float64
}

// TODO Fix Name
type SomeSource struct {
	Type        string
	CritChance  float64
	PhysicalMin *float64
	PhysicalMax *float64
	AttackRate  *float64
}

type Out string

const (
	OutHexDoomLimit       = Out("HexDoomLimit")
	OutBonechillDotEffect = Out("BonechillDotEffect")
	OutBonechillEffect    = Out("BonechillEffect")
)

type OutTable string

const (
	OutTableMainHand = OutTable("MainHand")
	OutTableOffHand  = OutTable("OffHand")
)

type ActiveSkill struct {
	SkillFlags       map[SkillFlag]bool
	SkillModList     *ModList
	SkillCfg         *ListCfg
	SkillTypes       map[data.SkillType]bool
	SkillData        map[string]interface{} // TODO Implement. Might be SkillData?
	ActiveEffect     *ActiveEffect
	Weapon1Cfg       *ListCfg
	Weapon2Cfg       *ListCfg
	SupportList      []interface{}
	Actor            *Actor `json:"-"`
	SocketGroup      interface{}
	SummonSkill      interface{}
	ConversionTable  map[data.DamageType]ConversionTable
	Minion           interface{}
	Weapon1Flags     data.ModFlag
	Weapon2Flags     data.ModFlag
	EffectList       []*ActiveEffect
	DisableReason    string
	BaseSkillModList *ModList
}

type ConversionTable struct {
	Targets map[data.DamageType]float64
	Mult    float64
}

type SkillFlag string

const (
	SkillFlagBrand            = SkillFlag("brand")
	SkillFlagHex              = SkillFlag("hex")
	SkillFlagCurse            = SkillFlag("curse")
	SkillFlagAttack           = SkillFlag("attack")
	SkillFlagWeapon1Attack    = SkillFlag("weapon1Attack")
	SkillFlagWeapon2Attack    = SkillFlag("weapon2Attack")
	SkillFlagSelfCast         = SkillFlag("selfCast")
	SkillFlagNotAverage       = SkillFlag("notAverage")
	SkillFlagShowAverage      = SkillFlag("showAverage")
	SkillFlagHit              = SkillFlag("hit")
	SkillFlagProjectile       = SkillFlag("projectile")
	SkillFlagTrap             = SkillFlag("trap")
	SkillFlagMine             = SkillFlag("mine")
	SkillFlagTotem            = SkillFlag("totem")
	SkillFlagBothWeaponAttack = SkillFlag("bothWeaponAttack")
	SkillFlagBuffs            = SkillFlag("buffs")
	SkillFlagCombat           = SkillFlag("combat")
	SkillFlagEffective        = SkillFlag("effective")
	SkillFlagSpell            = SkillFlag("spell")
	SkillFlagMelee            = SkillFlag("melee")
	SkillFlagChaining         = SkillFlag("chaining")
	SkillFlagArea             = SkillFlag("area")
	SkillFlagCast             = SkillFlag("cast")
	SkillFlagShieldAttack     = SkillFlag("shieldAttack")
	SkillFlagForceMainHand    = SkillFlag("forceMainHand")
	SkillFlagDisable          = SkillFlag("disable")
)

type SkillData struct {
	SupportBonechill      bool
	Cooldown              float64
	Triggered             bool
	TriggeredByBrand      bool
	TriggeredOnDeath      bool
	TriggerTime           *float64
	TriggeredBySaviour    bool
	CritChance            *float64
	SetOffHandPhysicalMin *float64
	SetOffHandPhysicalMax *float64
	AttackTime            *float64
	CastTimeOverride      *float64
	TimeOverride          *float64
	FixedCastTime         bool
	TriggerRate           *float64
	ShowAverage           bool
}

type ActiveEffect struct {
	GrantedEffect GrantedEffect
}

type GrantedEffect struct {
	Name                string
	CastTime            *float64
	Parts               []interface{}
	SkillTypes          map[data.SkillType]bool
	BaseFlags           map[SkillFlag]bool
	WeaponTypes         []data.WeaponRestriction
	Support             bool
	BaseMultiplier      *float64
	DamageEffectiveness *float64
}

type DamagePass struct {
	Label     string
	Source    map[string]interface{}
	Config    *ListCfg
	Output    map[string]float64
	Breakdown interface{} // TODO Implement Breakdown
}
