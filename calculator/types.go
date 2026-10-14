package calculator

import (
	"sync"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type Calculator struct {
	PoB *pob.PathOfBuilding

	AlignCalcsSkillToMain bool

	main *Environment
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
	//pob:src/Modules/Calcs.lua:125-135
	OutputModeCalculator = OutputMode("CALCULATOR")
)

type BuffMode string

const (
	BuffModeEffective = BuffMode("EFFECTIVE")
	BuffModeCombat    = BuffMode("COMBAT")
	BuffModeBuffed    = BuffMode("BUFFED")
	BuffModeUnbuffed  = BuffMode("UNBUFFED")
)

type Environment struct {
	Cache           *EnvironmentCache
	Build           *pob.PathOfBuilding
	Mode            OutputMode
	Spec            *PassiveSpec
	override        *CalcOverride
	itemText        map[int]string
	clusterNodes    map[string]data.Node
	displayEffects  map[[2]int]*GemEffect
	itemSupports    map[string]bool
	itemsByName     map[string]*Item
	JewelRadiusData map[string]map[string]float64
	partyExports    *partyBuffExports

	ModDB      *moddb.ModDB
	EnemyModDB *moddb.ModDB
	ItemModDB  *moddb.ModDB

	EnemyLevel int

	ConfigPlaceholder map[string]float64

	Player *Actor
	Enemy  *Actor
	Minion *ActorMinion

	RequirementsTable      []*RequirementsTable
	RequirementsTableItems []*RequirementsTable
	RequirementsTableGems  []*RequirementsTable

	RadiusJewelList     map[string]interface{}
	ExtraRadiusNodeList map[string]interface{}
	GrantedSkills       map[string]interface{}
	GrantedSkillsNodes  map[string]interface{}
	GrantedSkillsItems  map[string]interface{}
	Flasks              []*Item
	Tinctures           []*Item

	GrantedPassives map[string]interface{}
	AllocatedNodes  map[string]data.Node

	GrantedAscendancyNodes map[string]bool

	ItemClassRestrictions map[string]string

	AuxSkillList []*ActiveSkill

	Buffs       map[string]*moddb.ModList
	Debuffs     map[string]*moddb.ModList
	MinionBuffs map[string]*moddb.ModList
	curseSlots  []*curseEntry

	partyTab *partyTab

	ModeBuffs     bool
	ModeCombat    bool
	ModeEffective bool

	KeystonesAdded  map[string]interface{}
	MainSocketGroup int

	SkillsUsed           map[string]bool
	ConditionsUsed       map[string][]mod.Mod
	MultipliersUsed      map[string][]mod.Mod
	MinionConditionsUsed map[string][]mod.Mod
	EnemyConditionsUsed  map[string][]mod.Mod
	EnemyMultipliersUsed map[string][]mod.Mod

	PerStatsUsed      map[string][]mod.Mod
	EnemyPerStatsUsed map[string][]mod.Mod
	ModsUsed          map[string][]mod.Mod
	TagTypesUsed      map[string][]mod.Mod

	DebugErrors  []string
	CalcProps    map[string]float64
	AegisModList *moddb.ModList
	TheIronMass  *moddb.ModList

	TracePoolWalk bool
	PoolWalkTrace []PoolWalkStep

	InitialNodeModDB *moddb.ModList

	WeaponModList1 *moddb.ModList

	triggerCache *triggerSkillCache
	skipEHP      bool

	fullDPSSkillCosts map[int]skillCostEntry

	spare spareOutputs

	specAllocCounts allocatedNodeCounts

	nodeModList *moddb.ModList
	accelerated bool

	guardCandidates     []string
	allyCurseCandidates []string
	abyssStatTies       map[string]abyssStatTie
}

type EnvironmentCache struct {
	mu sync.RWMutex

	TreeVersion  pob.TreeVersion
	modsForNodes map[string]moddb.ModList

	items map[string]*Item

	guardPick  string
	pairsPicks map[string][]string
}

type Actor struct {
	ModDB                    *moddb.ModDB
	Level                    int
	Enemy                    *Actor `json:"-"`
	Player                   *Actor `json:"-"`
	Parent                   *Actor `json:"-"`
	partyMembers             *partyActor
	allyLifeList             map[string][]*allyLifeEntry
	MinionData               *raw.MinionInfo
	ItemList                 map[string]*ItemData
	ActiveSkillList          []*ActiveSkill
	Output                   map[string]float64
	OutputTable              map[OutTable]map[string]float64
	OutputStrings            map[string]string
	SkillDPS                 []SkillDPS
	MainSkill                *ActiveSkill
	Breakdown                *Breakdown
	WeaponData1              *SkillData
	WeaponData2              *SkillData
	WeaponRange1             float64
	WeaponRange2             float64
	StrDmgBonus              float64
	DamageShiftTable         map[data.DamageType]map[data.DamageType]float64
	DamageOverTimeShiftTable map[data.DamageType]map[data.DamageType]float64
	appliedEnemyModifiers    map[mod.Mod]int
	breakdownModLists        map[string][]moddb.ModResult
}

type ActorMinion struct {
	*Actor

	Type      string
	LifeTable []float64
	Uses      map[string]bool
	ItemSet   *pob.ItemSet
}

func (a *Actor) GetOutput(stat string) (float64, bool) {
	v, ok := a.Output[stat]
	return v, ok
}

var _ moddb.MinionMonsterTags = (*Actor)(nil)

func (a *Actor) MonsterTags() ([]string, bool) {
	if a == nil || a.MinionData == nil || a.MinionData.MonsterTags == nil {
		return nil, false
	}
	return a.MinionData.MonsterTags, true
}

var _ moddb.ActorLinks = (*Actor)(nil)

//pob:block:src/Classes/ModStore.lua:592
func (a *Actor) AddedUsingCondition(v string) (bool, bool) {
	allOneH := (*SkillData)(nil)
	switch {
	case a.WeaponData1 != nil && a.WeaponData1.CountsAsAll1H:
		allOneH = a.WeaponData1
	case a.WeaponData2 != nil && a.WeaponData2.CountsAsAll1H:
		allOneH = a.WeaponData2
	}
	if allOneH == nil {
		return false, false
	}
	var f *bool
	switch v {
	case "UsingAxe":
		f = allOneH.AddedUsingAxe
	case "UsingSword":
		f = allOneH.AddedUsingSword
	case "UsingDagger":
		f = allOneH.AddedUsingDagger
	case "UsingMace":
		f = allOneH.AddedUsingMace
	case "UsingClaw":
		f = allOneH.AddedUsingClaw
	}
	if f == nil {
		return false, false
	}
	return *f, true
}

func (a *Actor) RelatedModDB(name string) moddb.ModStoreFuncs {
	var related *Actor
	switch name {
	case "enemy":
		related = a.Enemy
	case "parent":
		related = a.Parent
	case "player":
		//pob:block:src/Classes/ModStore.lua:37-43
		switch {
		case a.Player != nil:
			related = a.Player
		case a.Parent != nil && a.Parent.Player != nil:
			related = a.Parent.Player
		case a.Enemy != nil && a.Enemy.Player != nil:
			related = a.Enemy.Player
		}
	case "partyMembers":
		if a.partyMembers == nil {
			return nil
		}
		return a.partyMembers.modDB
	}

	if related == nil || related.ModDB == nil {
		return nil
	}
	return related.ModDB
}

var _ moddb.ReservingAuras = (*Actor)(nil)

//pob:block:src/Classes/ModStore.lua:283-289
func (a *Actor) reservingSkillData(skillName, summonSkillName string) (*SkillData, bool) {
	for _, activeSkill := range a.ActiveSkillList {
		if activeSkill == nil || !activeSkill.SkillTypes[data.SkillTypeHasReservation] || activeSkill.SkillFlags[SkillFlagDisable] {
			continue
		}
		if len(activeSkill.BuffList) == 0 {
			continue
		}
		if !buffListHasName(activeSkill.BuffList, skillName) && !buffListHasName(activeSkill.BuffList, summonSkillName) {
			continue
		}
		return activeSkill.SkillData, true
	}
	return nil, false
}

func buffListHasName(buffList []*Buff, name string) bool {
	if name == "" {
		return false
	}
	for _, buff := range buffList {
		if buff != nil && buff.Name == name {
			return true
		}
	}
	return false
}

func (a *Actor) AuraManaReservedBase(skillName, summonSkillName string) (float64, bool) {
	skillData, found := a.reservingSkillData(skillName, summonSkillName)
	if !found {
		return 0, false
	}
	if skillData == nil {
		return 0, true
	}
	return skillData.ManaReservedBase, true
}

func (a *Actor) AuraLifeReservedBase(skillName, summonSkillName string) (float64, bool) {
	skillData, found := a.reservingSkillData(skillName, summonSkillName)
	if !found {
		return 0, false
	}
	if skillData == nil {
		return 0, true
	}
	return skillData.LifeReservedBase, true
}

type ItemData struct {
	ArmourData  *ArmourData
	Type        string
	Name        string
	ModList     *moddb.ModList
	SlotModList map[int]*moddb.ModList
	WeaponData  []interface{}

	Rarity    string
	Corrupted bool

	SocketColors              []string
	SocketColourAlwaysMatches bool

	ExplicitModLines []string
	OtherModLines    []string

	GrantedSkills []ItemGrantedSkill

	shaper, elder bool
}

type ArmourData struct {
	Ward         float64
	EnergyShield float64
	Armour       float64
	Evasion      float64
	BlockChance  float64
}

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
	SkillFlags         map[SkillFlag]bool
	SkillModList       *moddb.ModList
	SkillCfg           *moddb.ListCfg
	SkillTypes         map[data.SkillType]bool
	SkillData          *SkillData
	ActiveEffect       *GemEffect
	Weapon1Cfg         *moddb.ListCfg
	Weapon2Cfg         *moddb.ListCfg
	SupportList        []*GemEffect
	TriggeredBy        *GemEffect
	Actor              *Actor `json:"-"`
	SocketGroup        interface{}
	SocketGroupIndex   int
	SummonSkill        *ActiveSkill
	ConversionTable    map[data.DamageType]ConversionTable
	CfgConversionTable map[*moddb.ListCfg]map[data.DamageType]ConversionTable
	Minion             *ActorMinion
	MinionList         []string
	Weapon1Flags       mod.MFlag
	Weapon2Flags       mod.MFlag
	EffectList         []*GemEffect
	DisableReason      string
	BaseSkillModList   *moddb.ModList
	SlotName           string
	MinionSkillTypes   map[data.SkillType]bool
	BleedCfg           *moddb.ListCfg
	OHBleedCfg         *moddb.ListCfg
	PoisonCfg          *moddb.ListCfg
	OHPoisonCfg        *moddb.ListCfg
	IgniteCfg          *moddb.ListCfg
	OHIgniteCfg        *moddb.ListCfg
	DecayCfg           *moddb.ListCfg
	DotCfg             *moddb.ListCfg
	DotTypeCfg         map[data.DamageType]*moddb.ListCfg
	SkillTotemId       int
	SkillPart          int
	SkillPartName      string
	ActiveMineCount    *float64
	ActiveStageCount   float64
	ExtraSkillModList  []mod.Mod
	BuffList           []*Buff
	BuffSkill          bool
	MinionBuffSkill    bool
	DebuffSkill        bool
	Mirage             *MirageResult
	Marked             bool
	mirageCopy         bool

	infoMessage  string
	infoMessage2 string
	infoTrigger  string

	reusableMinionSkills []*ActiveSkill
	reusableMinionMain   *ActiveSkill

	areaOfEffectRadiusTertiaryOccurrences map[float64]float64
}

type MirageResult struct {
	Name          string
	Count         float64
	SkillPartName string
	Output        map[string]float64
}

type Buff struct {
	Type              string
	Name              string
	ActiveSkillBuff   bool
	ApplyNotPlayer    bool
	ApplyMinions      bool
	ApplyAllies       bool
	AllowTotemBuff    bool
	Cond              string
	EnemyCond         string
	StackVar          string
	StackLimitVar     string
	StackLimit        int
	ModList           *moddb.ModList
	UnscalableModList *moddb.ModList
}

type ConversionTable struct {
	Targets    map[data.DamageType]float64
	Mult       float64
	Conversion map[data.DamageType]float64
	Gain       map[data.DamageType]float64
}

type SkillFlag string

const (
	SkillFlagBrand            = SkillFlag("brand")
	SkillFlagArrow            = SkillFlag("arrow")
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
	SkillFlagMonsterExplode   = SkillFlag("monsterExplode")
	SkillFlagCast             = SkillFlag("cast")
	SkillFlagShieldAttack     = SkillFlag("shieldAttack")
	SkillFlagForceMainHand    = SkillFlag("forceMainHand")
	SkillFlagDisable          = SkillFlag("disable")
	SkillFlagBleed            = SkillFlag("bleed")
	SkillFlagDuration         = SkillFlag("duration")
	SkillFlagIgniteCanStack   = SkillFlag("igniteCanStack")
	SkillFlagIgniteToChaos    = SkillFlag("igniteToChaos")
	SkillFlagDotCanStack      = SkillFlag("DotCanStack")
	SkillFlagDot              = SkillFlag("dot")
	SkillFlagDotFromAttack    = SkillFlag("dotFromAttack")
	SkillFlagIgnite           = SkillFlag("ignite")
	SkillFlagPoison           = SkillFlag("poison")
	SkillFlagDecay            = SkillFlag("decay")
	SkillFlagImpale           = SkillFlag("impale")
	SkillFlagBallista         = SkillFlag("ballista")
	SkillFlagMinion           = SkillFlag("minion")
	SkillFlagPermanentMinion  = SkillFlag("permanentMinion")
	SkillFlagRandomPhys       = SkillFlag("randomPhys")
	SkillFlagForking          = SkillFlag("forking")
	SkillFlagPiercing         = SkillFlag("piercing")
	SkillFlagWarcry           = SkillFlag("warcry")
	SkillFlagHaveMinion       = SkillFlag("haveMinion")
	SkillFlagMinionSkill      = SkillFlag("minionSkill")
	SkillFlagSpectre          = SkillFlag("spectre")
	SkillFlagStages           = SkillFlag("stages")
	SkillFlagMultiPart        = SkillFlag("multiPart")
	SkillFlagMultiStage       = SkillFlag("multiStage")
	SkillFlagTriggered        = SkillFlag("triggered")
	SkillFlagFocused          = SkillFlag("focused")
)

type SkillData struct {
	StagesMax                               float64
	PulseInterval                           float64
	PulseDamage                             float64
	SupportBonechill                        bool
	Cooldown                                float64
	StoredUses                              *float64
	Triggered                               bool
	TriggeredByBrand                        bool
	TriggeredByBattleMageCry                bool
	TriggerOnCrit                           bool
	CountsAttachedBrandsInDamage            bool
	AttachedBrandCount                      float64
	TriggeredOnDeath                        bool
	TriggeredByAutoexertion                 bool
	TriggeredByDamageTaken                  float64
	TriggeredWhileChannelling               bool
	TriggeredByMeleeKill                    bool
	TriggeredByCospris                      bool
	TriggeredByMjolner                      bool
	TriggeredBySevenTeachings               bool
	TriggeredByKineticFlux                  bool
	TriggeredByUnique                       bool
	TriggeredByFocus                        bool
	TriggeredByCraft                        bool
	TriggeredByManaSpent                    bool
	TriggeredByManaforged                   *float64
	ManaForgedArrowsPercentThreshold        float64
	TriggeredByParentAttack                 bool
	TriggeredByGeneralsCry                  bool
	TriggerTime                             float64
	TriggeredBySaviour                      bool
	CritChance                              float64
	SetOffHandPhysicalMin                   float64
	SetOffHandPhysicalMax                   float64
	PercentageLifeSacrificed                float64
	PercentageSacrificedDealtAsChaos        float64
	AdditionalRuinChance                    float64
	ManabondMissingManaGainPercent          float64
	ManabondMissingUnreservedManaPercentage float64
	SetOffHandFireMin                       float64
	SetOffHandFireMax                       float64
	SetOffHandColdMin                       float64
	SetOffHandColdMax                       float64
	AttackTime                              float64
	CastTimeOverride                        float64
	CastTimeOverridesAttackTime             bool
	TimeOverride                            float64
	FixedCastTime                           bool
	TriggerRate                             *float64
	ShowAverage                             bool
	ManaReservationPercent                  *float64
	TotemLevel                              int
	CannotBeEvaded                          bool
	DoubleHitsWhenDualWielding              bool
	DpsMultiplier                           float64
	DpsBaseMultiplier                       float64
	PodOverlapMultiplier                    float64
	DamagePerProjectile                     float64
	DelayPerProjectile                      float64
	MaxRageVortexSacrificePercentage        float64
	FifthStrikeDamage                       float64
	FifthStrikeProjectiles                  float64
	BaseMultiplier                          float64
	DamageEffectiveness                     float64
	LifeLeechPerUse                         float64
	ManaLeechPerUse                         float64
	BleedDurationIsSkillDuration            bool
	BleedIsSkillEffect                      bool
	Duration                                float64
	FrostBombDamagePer100ms                 float64
	BleedBasePercent                        float64
	Type                                    data.ItemClassName
	AttackRate                              float64
	PhysicalMin                             float64
	PhysicalMax                             float64
	AttackSpeedInc                          float64
	CountsAsAll1H                           bool
	CountsAsDualWielding                    bool

	AddedUsingAxe    *bool
	AddedUsingSword  *bool
	AddedUsingDagger *bool
	AddedUsingMace   *bool
	AddedUsingClaw   *bool

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

	LifeDealtAsChaos         float64
	EnergyShieldDealtAsChaos float64
	LifeDealtAsFire          float64
	triggerSource            *cachedTriggerSkill
	SkeletonLife             float64
	SelfDamageTakenLife      float64
	SelfDamageTakenES        float64

	Radius                                     float64
	RadiusSecondary                            float64
	RadiusTertiary                             float64
	RadiusLabel                                string
	RadiusSecondaryLabel                       string
	RadiusTertiaryLabel                        string
	RadiusExtra                                float64
	RadiusTertiaryBaseMargin                   float64
	ProjectileSpeedAppliesToMSAreaOfEffect     bool
	DurationSecondary                          float64
	DurationTertiary                           float64
	MinionLevel                                float64
	MinionLevelIsEnemyLevel                    bool
	MinionLevelIsPlayerLevel                   float64
	MinionDamageEffectiveness                  float64
	FireDot                                    float64
	RFLifeMultiplier                           float64
	RFESMultiplier                             float64
	RFManaMultiplier                           float64
	RFMultiplier                               float64
	PercentSacrificed                          float64
	TriggeredByCoC                             bool
	DotIsSpell                                 bool
	RepeatFrequency                            float64
	RepeatInterval                             float64
	HitFrequency                               float64
	CremationFireRate                          float64
	CremationFireRateIncrease                  float64
	DamageInterval                             float64
	HitFrequencyPerBlade                       float64
	IncVolleyFrequency                         float64
	VaalBladeVortexBlade                       float64
	RepeatFrequencyIncrease                    float64
	BeamOverlapMultiplier                      float64
	ActiveArrowMultiplier                      *float64
	TornadoShotSecondaryHitChance              *float64
	ShrapnelBallistaProjectileOverlap          *float64
	HoaOverkill                                *float64
	HoaMoreBurn                                float64
	HoaOverkillPercent                         float64
	MoltenShellDamageMitigated                 *float64
	VaalMoltenShellDamageMitigated             *float64
	MoltenShellReflect                         float64
	TriggeredByMirageArcher                    bool
	TriggeredBySnipe                           bool
	TriggeredBySacredWisps                     bool
	ChanceToTriggerOnCrit                      float64
	ColdDot                                    float64
	CorpseExplosionLifeMultiplier              float64
	SelfFireExplosionLifeMultiplier            float64
	CorpseLife                                 float64
	CorpseExplosionDamageType                  string
	ChaosDot                                   float64
	BaseManaCostIsAtLeastPercentUnreservedMana float64
	ManaReservationFlat                        *float64
	LifeReservationFlat                        *float64
	LifeReservationPercent                     *float64
	ManaReservationFlatForced                  *float64
	LifeReservationFlatForced                  *float64
	ManaReservationPercentForced               *float64
	LifeReservationPercentForced               *float64
	ManaReservedBase                           float64
	ManaReservedPercent                        float64
	LifeReservedBase                           float64
	LifeReservedPercent                        float64
	AttackSpeedMultiplier                      float64
	MinionUseBowAndQuiver                      bool
	MinionUseMainHandWeapon                    bool
	ArrowSpeedAppliesToAreaOfEffect            bool
	GainPercentBaseWandDamageToSpells          float64
	HitTimeOverride                            float64
	TrapCooldown                               float64
	MineDurationAppliesToSkill                 bool
	Debuff                                     bool
	DebuffSecondary                            bool
	DebuffTertiary                             bool
	ReserveDuration                            float64
	AuraDuration                               float64

	SoulPreventionDuration                 float64
	SkillEffectAppliesToSoulGainPrevention bool

	AuraCannotAffectSelf bool
	ThisIsNotABuff       bool
	BuffNotPlayer        bool
	BuffMinions          bool
	BuffAllies           bool
	AllowTotemBuff       bool
	Enable               bool

	DotIsArea                           bool
	DotIsProjectile                     bool
	DotIsAttack                         bool
	DotIsHit                            bool
	PhysicalDot                         float64
	LightningDot                        float64
	ExplodeCorpse                       bool
	HitChanceIsExplodeChance            bool
	PhysicalEffectiveExplodePercentage  float64
	LightningEffectiveExplodePercentage float64
	ColdEffectiveExplodePercentage      float64
	FireEffectiveExplodePercentage      float64
	ChaosEffectiveExplodePercentage     float64
	PoisonIsSkillEffect                 bool
	PoisonDurationIsSkillDuration       bool
	StackMultiplier                     float64
	RadiusIsWeaponRange                 bool
	Decay                               float64
	BaseStunDuration                    float64
	Range                               float64
	RangeBonus                          float64
	HitTimeMultiplier                   float64
	TotemFireOnce                       bool
	NumStages                           float64
	ProjectileSpeed                     float64
	StackCount                          *float64
	SummonSpeed                         *float64
	StrikeInterval                      float64
	RemainingDurationDamage             float64
}

type GrantedEffect struct {
	Raw        *poe.GrantedEffect
	Parts      []raw.SkillPart
	SkillTypes map[data.SkillType]bool
	Funcs      map[string]func(activeSkill *ActiveSkill, output map[string]float64, breakdown *Breakdown)

	fromItem     bool
	itemSupports map[string]bool
}

func (g *GrantedEffect) ID() string {
	if g.Raw == nil {
		return ""
	}
	return g.Raw.ID
}

func (g *GrantedEffect) Name() string {
	if g.Raw == nil {
		return ""
	}
	if pobSkill, ok := raw.PoBSkills[g.Raw.ID]; ok && pobSkill.Name != "" {
		return pobSkill.Name
	}
	if as := g.Raw.GetActiveSkill(); as != nil && as.DisplayedName != "" {
		return as.DisplayedName
	}
	return g.Raw.ID
}

//pob:block:src/Modules/CalcActiveSkill.lua:287
func (g *GrantedEffect) WeaponTypes() []data.ItemClassName {
	id := g.ID()
	if alias, aliased := skillFlagIDAliases[id]; aliased {
		id = alias
	}

	names := raw.SkillWeaponTypes[id]
	if names == nil {
		return nil
	}

	out := make([]data.ItemClassName, len(names))
	for i, name := range names {
		out[i] = data.ItemClassName(name)
	}
	return out
}

func (g *GrantedEffect) FromItem() bool {
	if g == nil || g.Raw == nil {
		return false
	}
	return g.fromItem || g.itemSupports[g.Raw.ID] || raw.PoBSkills[g.Raw.ID].FromItem
}

func (g *GrantedEffect) IsSupport() bool {
	if g == nil || g.Raw == nil {
		return false
	}
	return raw.PoBSkills[g.Raw.ID].Support
}

func (g *GrantedEffect) BaseMultiplier() float64 {
	return 0
}

func (g *GrantedEffect) DamageEffectiveness() float64 {
	return 0
}

func (g *GrantedEffect) CastTime() float64 {
	return float64(g.Raw.CastTime) / 1000
}

type DamagePass struct {
	Label     string
	Source    *SkillData
	Config    *moddb.ListCfg
	Output    map[string]float64
	Breakdown *Breakdown
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

	SocketQuality   int
	ItemQuality     int
	GlobalQuality   int
	SupportQuality  int
	req             *int
	triggerChance   *float64
	gemCfg          *moddb.ListCfg
	gemPropertyInfo []gemPropertyValue

	ActorLevel *int

	GrantedEffectLevel *raw.CalculatedLevel

	CostEmptied bool

	ExplodeSource *ExplodeSource

	Superseded   bool
	IsSupporting map[*pob.Gem]bool
	Values       map[string]float64

	MainSkill          *ActiveSkill
	ActivationFreqInc  float64
	ActivationFreqMore float64
	AttachedBrandCount float64
	IgnoresTickRate    bool
}

type PoolWalkStep struct {
	N               int
	Damage          map[data.DamageType]float64
	LifeIn          float64
	ManaIn          float64
	EnergyShieldIn  float64
	WardIn          float64
	LifeOut         float64
	ManaOut         float64
	EnergyShieldOut float64
	WardOut         float64
	RecoupableIn    map[data.DamageType]float64
	RecoupableOut   map[data.DamageType]float64

	AegisIn  map[string]float64
	AegisOut map[string]float64
	GuardIn  map[string]float64
	GuardOut map[string]float64
	AllyIn   map[string]float64
	AllyOut  map[string]float64

	Actor *Actor
}
