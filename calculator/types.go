package calculator

import (
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
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

	ModDB      *moddb.ModDB
	EnemyModDB *moddb.ModDB
	ItemModDB  *moddb.ModDB

	EnemyLevel int

	Player *Actor
	Enemy  *Actor
	Minion *ActorMinion

	RequirementsTable      []*RequirementsTable
	RequirementsTableItems []*RequirementsTable
	RequirementsTableGems  []*RequirementsTable

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

	DebugErrors  []string
	CalcProps    map[string]float64
	AegisModList *moddb.ModList
	TheIronMass  *moddb.ModList
}

type EnvironmentCache struct {
	TreeVersion  pob.TreeVersion
	modsForNodes map[string]moddb.ModList // Mods for all nodes cached after being parsed
}

type Actor struct {
	ModDB                *moddb.ModDB
	Level                int
	Enemy                *Actor `json:"-"`
	ItemList             map[string]*ItemData
	ActiveSkillList      []*ActiveSkill
	Output               map[string]float64
	OutputTable          map[OutTable]map[string]float64
	OutputStrings        map[string]string
	MainSkill            *ActiveSkill
	Breakdown            *Breakdown
	WeaponData1          *SkillData
	WeaponData2          *SkillData
	StrDmgBonus          float64
	Reserved_LifeBase    float64
	Reserved_LifePercent float64
	Reserved_ManaBase    float64
	Reserved_ManaPercent float64
	DamageShiftTable     map[data.DamageType]map[data.DamageType]float64
}

type ActorMinion struct {
	*Actor

	Type       string
	MinionData MinionData
	LifeTable  map[int]int
}

type MinionData struct {
	Life            float64
	EnergyShield    float64
	Armour          float64
	FireResist      float64
	ColdResist      float64
	LightningResist float64
	ChaosResist     float64
	Accuracy        float64
	ModList         []mod.Mod
	Limit           string
}

func (a *Actor) GetOutput(stat string) (float64, bool) {
	v, ok := a.Output[stat]
	return v, ok
}

type ItemData struct {
	ArmourData  *ArmourData
	Type        string
	ModList     *moddb.ModList
	SlotModList map[int]*moddb.ModList
	WeaponData  []interface{}
}

type ArmourData struct {
	Ward         float64
	EnergyShield float64
	Armour       float64
	Evasion      float64
	BlockChance  float64
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
	SkillFlags        map[SkillFlag]bool
	SkillModList      *moddb.ModList
	SkillCfg          *moddb.ListCfg
	SkillTypes        map[data.SkillType]bool
	SkillData         *SkillData
	ActiveEffect      *GemEffect
	Weapon1Cfg        *moddb.ListCfg
	Weapon2Cfg        *moddb.ListCfg
	SupportList       []*GemEffect
	Actor             *Actor `json:"-"`
	SocketGroup       interface{}
	SummonSkill       *ActiveSkill
	ConversionTable   map[data.DamageType]ConversionTable
	Minion            *ActorMinion
	Weapon1Flags      mod.MFlag
	Weapon2Flags      mod.MFlag
	EffectList        []*GemEffect
	DisableReason     string
	BaseSkillModList  *moddb.ModList
	SlotName          string
	MinionSkillTypes  map[data.SkillType]bool
	BleedCfg          *moddb.ListCfg
	OHBleedCfg        *moddb.ListCfg
	SkillTotemId      int
	SkillPartName     string
	ActiveMineCount   float64
	ExtraSkillModList []mod.Mod
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
	SkillFlagDot              = SkillFlag("dot")
	SkillFlagIgnite           = SkillFlag("ignite")
	SkillFlagDecay            = SkillFlag("decay")
	SkillFlagImpale           = SkillFlag("impale")
	SkillFlagBallista         = SkillFlag("ballista")
	SkillFlagMinion           = SkillFlag("minion")
	SkillFlagRandomPhys       = SkillFlag("randomPhys")
	SkillFlagForking          = SkillFlag("forking")
	SkillFlagPiercing         = SkillFlag("piercing")
	SkillFlagWarcry           = SkillFlag("warcry")
)

type SkillData struct {
	SupportBonechill             bool
	Cooldown                     float64
	Triggered                    bool
	TriggeredByBrand             bool
	TriggeredOnDeath             bool
	TriggerTime                  float64
	TriggeredBySaviour           bool
	CritChance                   float64
	SetOffHandPhysicalMin        float64
	SetOffHandPhysicalMax        float64
	AttackTime                   float64
	CastTimeOverride             float64
	TimeOverride                 float64
	FixedCastTime                bool
	TriggerRate                  float64
	ShowAverage                  bool
	ManaReservationPercent       float64
	TotemLevel                   int
	CannotBeEvaded               bool
	DoubleHitsWhenDualWielding   bool
	DpsMultiplier                float64
	BaseMultiplier               float64
	DamageEffectiveness          float64
	LifeLeechPerUse              float64
	ManaLeechPerUse              float64
	BleedDurationIsSkillDuration bool
	BleedIsSkillEffect           bool
	Duration                     float64
	BleedBasePercent             float64
	Type                         data.ItemClassName
	AttackRate                   float64
	PhysicalMin                  float64
	PhysicalMax                  float64
	AttackSpeedInc               float64
	CountsAsAll1H                bool
	CountsAsDualWielding         bool

	PhysicalBonusMin  float64
	PhysicalBonusMax  float64
	LightningMin      float64
	LightningMax      float64
	LightningBonusMin float64
	LightningBonusMax float64
	ColdMin           float64
	ColdMax           float64
	ColdBonusMin      float64
	ColdBonusMax      float64
	FireMin           float64
	FireMax           float64
	FireBonusMin      float64
	FireBonusMax      float64
	ChaosMin          float64
	ChaosMax          float64
	ChaosBonusMin     float64
	ChaosBonusMax     float64

	RadiusExtra                                float64
	DurationSecondary                          float64
	MinionLevel                                float64
	FireDot                                    float64
	TriggeredByCoC                             bool
	DotIsSpell                                 bool
	RepeatFrequency                            float64
	TriggeredByMirageArcher                    bool
	ChanceToTriggerOnCrit                      bool
	ColdDot                                    bool
	CorpseExplosionLifeMultiplier              float64
	ChaosDot                                   bool
	BaseManaCostIsAtLeastPercentUnreservedMana float64
	ManaReservationFlat                        float64
	LifeReservationFlat                        float64
	LifeReservationPercent                     float64
	ManaReservationFlatForced                  *float64
	LifeReservationFlatForced                  *float64
	ManaReservationPercentForced               *float64
	LifeReservationPercentForced               *float64
	ManaReservedBase                           float64
	ManaReservedPercent                        float64
	LifeReservedBase                           float64
	LifeReservedPercent                        float64
	MinionUseBowAndQuiver                      bool
	ArrowSpeedAppliesToAreaOfEffect            bool
	GainPercentBaseWandDamage                  float64
	HitTimeOverride                            float64
	TrapCooldown                               float64
	MineDurationAppliesToSkill                 bool
	Debuff                                     bool
	DebuffSecondary                            bool
	ReserveDuration                            float64
	AuraDuration                               float64
}

type GrantedEffect struct {
	Raw        *poe.GrantedEffect
	Parts      []interface{}
	SkillTypes map[data.SkillType]bool
	BaseFlags  map[SkillFlag]bool
	Funcs      map[string]func(activeSkill *ActiveSkill, output map[string]float64, breakdown *Breakdown)
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
	Source    *SkillData
	Config    *moddb.ListCfg
	Output    map[string]float64
	Breakdown interface{} // TODO Implement Breakdown
}

type RequirementsTable struct {
	Source     string
	SourceGem  *pob.Gem
	SourceItem interface{}
	Str        int
	Dex        int
	Int        int
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
