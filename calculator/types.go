package calculator

import (
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

type Calculator struct {
	PoB *pob.PathOfBuilding
}

func NewCalculator(build pob.PathOfBuilding) *Calculator {
	return &Calculator{
		PoB: &build,
	}
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
	Cache *EnvironmentCache
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

	RequirementsTableItems map[string]interface{}   // TODO Implement
	RequirementsTableGems  []*RequirementsTableGems // TODO Implement

	RadiusJewelList     map[string]interface{} // TODO Implement
	ExtraRadiusNodeList map[string]interface{} // TODO Implement
	GrantedSkills       map[string]interface{} // TODO Implement
	GrantedSkillsNodes  map[string]interface{} // TODO Implement
	GrantedSkillsItems  map[string]interface{} // TODO Implement
	Flasks              map[string]interface{} // TODO Implement

	GrantedPassives map[string]interface{} // TODO Implement
	AllocatedNodes  map[string]data.Node

	AuxSkillList map[string]interface{} // TODO Implement

	ModeBuffs     bool
	ModeCombat    bool
	ModeEffective bool

	KeystonesAdded  map[string]interface{}
	MainSocketGroup int

	DebugErrors []string
}

type EnvironmentCache struct {
	TreeVersion  data.TreeVersion
	modsForNodes map[string]ModList // Mods for all nodes cached after being parsed
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
	ActiveEffect     *GemEffect
	Weapon1Cfg       *ListCfg
	Weapon2Cfg       *ListCfg
	SupportList      []*GemEffect
	Actor            *Actor `json:"-"`
	SocketGroup      interface{}
	SummonSkill      *ActiveSkill
	ConversionTable  map[data.DamageType]ConversionTable
	Minion           interface{}
	Weapon1Flags     mod.MFlag
	Weapon2Flags     mod.MFlag
	EffectList       []*GemEffect
	DisableReason    string
	BaseSkillModList *ModList
	SlotName         string
	MinionSkillTypes map[data.SkillType]bool
	BleedCfg         *ListCfg
	OHBleedCfg       *ListCfg
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
	SkillFlagBleed            = SkillFlag("bleed")
	SkillFlagDuration         = SkillFlag("duration")
	SkillFlagIgniteCanStack   = SkillFlag("igniteCanStack")
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

type GrantedEffect struct {
	Raw        *poe.GrantedEffect
	Parts      []interface{}
	SkillTypes map[data.SkillType]bool
	BaseFlags  map[SkillFlag]bool
}

func (g *GrantedEffect) WeaponTypes() []data.ItemClassName {
	out := make([]data.ItemClassName, len(g.Raw.WeaponRestrictions))
	for i, restriction := range g.Raw.WeaponRestrictions {
		out[i] = data.ItemClassName(poe.ItemClasses[restriction].Name)
	}
	return out
}

func (g *GrantedEffect) BaseMultiplier() float64 {
	return 0 // TODO BaseMultiplier
}

func (g *GrantedEffect) DamageEffectiveness() float64 {
	return 0 // TODO DamageEffectiveness
}

func (g *GrantedEffect) CastTime() float64 {
	return float64(g.Raw.CastTime) / 1000
}

type DamagePass struct {
	Label     string
	Source    map[string]interface{}
	Config    *ListCfg
	Output    map[string]float64
	Breakdown interface{} // TODO Implement Breakdown
}

type RequirementsTableGems struct {
	Source    string
	SourceGem pob.Gem
	Str       int
	Dex       int
	Int       int
}

type GemEffect struct {
	GrantedEffect *GrantedEffect
	Level         int
	Quality       int
	QualityID     string
	SrcInstance   *pob.Gem
	GemData       *poe.SkillGem

	// For Active Gems
	GrantedEffectLevel *raw.CalculatedLevel

	// For Support Gems
	Superseded   bool
	IsSupporting map[*pob.Gem]bool
	Values       map[string]float64
}
