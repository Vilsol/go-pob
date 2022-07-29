package calculator

import (
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

	KeystonesAdded map[string]interface{}
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
	SkillFlags   map[SkillFlag]bool
	SkillModList *ModList
	SkillCfg     *ListCfg
	SkillTypes   map[SkillType]bool
	SkillData    map[string]interface{} // TODO Implement. Might be SkillData?
	ActiveEffect *ActiveEffect
	Weapon1Cfg   *ListCfg
	Weapon2Cfg   *ListCfg
	SupportList  []interface{}
	Actor        *Actor `json:"-"`
	SocketGroup  interface{}
	SummonSkill  interface{}
}

type SkillFlag string

const (
	SkillFlagBrand         = SkillFlag("brand")
	SkillFlagHex           = SkillFlag("hex")
	SkillFlagCurse         = SkillFlag("curse")
	SkillFlagAttack        = SkillFlag("attack")
	SkillFlagWeapon1Attack = SkillFlag("weapon1Attack")
	SkillFlagWeapon2Attack = SkillFlag("weapon2Attack")
	SkillFlagSelfCast      = SkillFlag("selfCast")
	SkillFlagNotAverage    = SkillFlag("notAverage")
	SkillFlagShowAverage   = SkillFlag("showAverage")
)

type SkillType int

const (
	SkillTypeAttack                         = SkillType(1)
	SkillTypeSpell                          = SkillType(2)
	SkillTypeProjectile                     = SkillType(3) // Specifically skills which fire projectiles
	SkillTypeDualWieldOnly                  = SkillType(4) // Attack requires dual wielding only used on Dual Strike
	SkillTypeBuff                           = SkillType(5)
	SkillTypeRemoved6                       = SkillType(6) // Now removed was CanDualWield: Attack can be used while dual wielding
	SkillTypeMainHandOnly                   = SkillType(7) // Attack only uses the main hand; removed in 3.5 but still needed for 2.6
	SkillTypeRemoved8                       = SkillType(8) // Now removed was only used on Cleave
	SkillTypeMinion                         = SkillType(9)
	SkillTypeDamage                         = SkillType(10) // Skill hits (not used on attacks because all of them hit)
	SkillTypeArea                           = SkillType(11)
	SkillTypeDuration                       = SkillType(12)
	SkillTypeRequiresShield                 = SkillType(13)
	SkillTypeProjectileSpeed                = SkillType(14)
	SkillTypeHasReservation                 = SkillType(15)
	SkillTypeReservationBecomesCost         = SkillType(16)
	SkillTypeTrappable                      = SkillType(17) // Skill can be turned into a trap
	SkillTypeTotemable                      = SkillType(18) // Skill can be turned into a totem
	SkillTypeMineable                       = SkillType(19) // Skill can be turned into a mine
	SkillTypeElementalStatus                = SkillType(20) // Causes elemental status effects but doesn't hit (used on Herald of Ash to allow Elemental Proliferation to apply)
	SkillTypeMinionsCanExplode              = SkillType(21)
	SkillTypeRemoved22                      = SkillType(22) // Now removed was AttackCanTotem
	SkillTypeChains                         = SkillType(23)
	SkillTypeMelee                          = SkillType(24)
	SkillTypeMeleeSingleTarget              = SkillType(25)
	SkillTypeMulticastable                  = SkillType(26) // Spell can repeat via Spell Echo
	SkillTypeTotemCastsAlone                = SkillType(27)
	SkillTypeMultistrikeable                = SkillType(28) // Attack can repeat via Multistrike
	SkillTypeCausesBurning                  = SkillType(29) // Deals burning damage
	SkillTypeSummonsTotem                   = SkillType(30)
	SkillTypeTotemCastsWhenNotDetached      = SkillType(31)
	SkillTypeFire                           = SkillType(32)
	SkillTypeCold                           = SkillType(33)
	SkillTypeLightning                      = SkillType(34)
	SkillTypeTriggerable                    = SkillType(35)
	SkillTypeTrapped                        = SkillType(36)
	SkillTypeMovement                       = SkillType(37)
	SkillTypeRemoved39                      = SkillType(38) // Now removed was Cast
	SkillTypeDamageOverTime                 = SkillType(39)
	SkillTypeRemoteMined                    = SkillType(40)
	SkillTypeTriggered                      = SkillType(41)
	SkillTypeVaal                           = SkillType(42)
	SkillTypeAura                           = SkillType(43)
	SkillTypeRemoved45                      = SkillType(44) // Now removed was LightningSpell
	SkillTypeCanTargetUnusableCorpse        = SkillType(45) // Doesn't appear to be used at all
	SkillTypeRemoved47                      = SkillType(46) // Now removed was TriggeredAttack
	SkillTypeRangedAttack                   = SkillType(47)
	SkillTypeRemoved49                      = SkillType(48) // Now removed was MinionSpell
	SkillTypeChaos                          = SkillType(49)
	SkillTypeFixedSpeedProjectile           = SkillType(50) // Not used by any skill
	SkillTypeRemoved52                      = SkillType(51)
	SkillTypeThresholdJewelArea             = SkillType(52) // Allows Burning Arrow and Vigilant Strike to be supported by Inc AoE and Conc Effect
	SkillTypeThresholdJewelProjectile       = SkillType(53)
	SkillTypeThresholdJewelDuration         = SkillType(54) // Allows Burning Arrow to be supported by Inc/Less Duration and Rapid Decay
	SkillTypeThresholdJewelRangedAttack     = SkillType(55)
	SkillTypeRemoved57                      = SkillType(56)
	SkillTypeChannel                        = SkillType(57)
	SkillTypeDegenOnlySpellDamage           = SkillType(58) // Allows Contagion Blight and Scorching Ray to be supported by Controlled Destruction
	SkillTypeRemoved60                      = SkillType(59) // Now removed was ColdSpell
	SkillTypeInbuiltTrigger                 = SkillType(60) // Skill granted by item that is automatically triggered prevents trigger gems and trap/mine/totem from applying
	SkillTypeGolem                          = SkillType(61)
	SkillTypeHerald                         = SkillType(62)
	SkillTypeAuraAffectsEnemies             = SkillType(63) // Used by Death Aura added by Blasphemy
	SkillTypeNoRuthless                     = SkillType(64)
	SkillTypeThresholdJewelSpellDamage      = SkillType(65)
	SkillTypeCascadable                     = SkillType(66) // Spell can cascade via Spell Cascade
	SkillTypeProjectilesFromUser            = SkillType(67) // Skill can be supported by Volley
	SkillTypeMirageArcherCanUse             = SkillType(68) // Skill can be supported by Mirage Archer
	SkillTypeProjectileSpiral               = SkillType(69) // Excludes Volley from Vaal Fireball and Vaal Spark
	SkillTypeSingleMainProjectile           = SkillType(70) // Excludes Volley from Spectral Shield Throw
	SkillTypeMinionsPersistWhenSkillRemoved = SkillType(71) // Excludes Summon Phantasm on Kill from Manifest Dancing Dervish
	SkillTypeProjectileNumber               = SkillType(72) // Allows LMP/GMP on Rain of Arrows and Toxic Rain
	SkillTypeWarcry                         = SkillType(73) // Warcry
	SkillTypeInstant                        = SkillType(74) // Instant cast skill
	SkillTypeBrand                          = SkillType(75)
	SkillTypeDestroysCorpse                 = SkillType(76) // Consumes corpses on use
	SkillTypeNonHitChill                    = SkillType(77)
	SkillTypeChillingArea                   = SkillType(78)
	SkillTypeAppliesCurse                   = SkillType(79)
	SkillTypeCanRapidFire                   = SkillType(80)
	SkillTypeAuraDuration                   = SkillType(81)
	SkillTypeAreaSpell                      = SkillType(82)
	SkillTypeOR                             = SkillType(83)
	SkillTypeAND                            = SkillType(84)
	SkillTypeNOT                            = SkillType(85)
	SkillTypePhysical                       = SkillType(86)
	SkillTypeAppliesMaim                    = SkillType(87)
	SkillTypeCreatesMinion                  = SkillType(88)
	SkillTypeGuard                          = SkillType(89)
	SkillTypeTravel                         = SkillType(90)
	SkillTypeBlink                          = SkillType(91)
	SkillTypeCanHaveBlessing                = SkillType(92)
	SkillTypeProjectilesNotFromUser         = SkillType(93)
	SkillTypeAttackInPlaceIsDefault         = SkillType(94)
	SkillTypeNova                           = SkillType(95)
	SkillTypeInstantNoRepeatWhenHeld        = SkillType(96)
	SkillTypeInstantShiftAttackForLeftMouse = SkillType(97)
	SkillTypeAuraNotOnCaster                = SkillType(98)
	SkillTypeBanner                         = SkillType(99)
	SkillTypeRain                           = SkillType(100)
	SkillTypeCooldown                       = SkillType(101)
	SkillTypeThresholdJewelChaining         = SkillType(102)
	SkillTypeSlam                           = SkillType(103)
	SkillTypeStance                         = SkillType(104)
	SkillTypeNonRepeatable                  = SkillType(105) // Blood and Sand + Flesh and Stone
	SkillTypeOtherThingUsesSkill            = SkillType(106)
	SkillTypeSteel                          = SkillType(107)
	SkillTypeHex                            = SkillType(108)
	SkillTypeMark                           = SkillType(109)
	SkillTypeAegis                          = SkillType(110)
	SkillTypeOrb                            = SkillType(111)
	SkillTypeKillNoDamageModifiers          = SkillType(112)
	SkillTypeRandomElement                  = SkillType(113) // means elements cannot repeat
	SkillTypeLateConsumeCooldown            = SkillType(114)
	SkillTypeArcane                         = SkillType(115) // means it is reliant on amount of mana spent
	SkillTypeFixedCastTime                  = SkillType(116)
	SkillTypeRequiresOffHandNotWeapon       = SkillType(117)
	SkillTypeLink                           = SkillType(118)
	SkillTypeBlessing                       = SkillType(119)
	SkillTypeZeroReservation                = SkillType(120)
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
	Name     string
	CastTime *float64
}

/*

	label = "Main Hand",
	source = source,
	cfg = activeSkill.weapon1Cfg,
	output = output.MainHand,
	breakdown = breakdown and breakdown.MainHand,
*/

type DamagePass struct {
	Label     string
	Source    map[string]interface{}
	Config    *ListCfg
	Output    map[string]float64
	Breakdown interface{} // TODO Implement
}
