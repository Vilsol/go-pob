package data

import (
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/mod"
)

type DamageType string

const (
	DamageTypePhysical  = DamageType("Physical")
	DamageTypeLightning = DamageType("Lightning")
	DamageTypeCold      = DamageType("Cold")
	DamageTypeFire      = DamageType("Fire")
	DamageTypeElemental = DamageType("Elemental")
	DamageTypeChaos     = DamageType("Chaos")
)

func (DamageType) Values() []DamageType {
	return []DamageType{
		DamageTypePhysical,
		DamageTypeLightning,
		DamageTypeCold,
		DamageTypeFire,
		DamageTypeChaos,
	}
}

func (t DamageType) IsElemental() bool {
	switch t {
	case DamageTypeFire:
		return true
	case DamageTypeLightning:
		return true
	case DamageTypeCold:
		return true
	default:
		return false
	}
}

var DamageTypeFlags = map[DamageType]int{
	DamageTypePhysical:  0x01,
	DamageTypeLightning: 0x02,
	DamageTypeCold:      0x04,
	DamageTypeFire:      0x08,
	DamageTypeElemental: 0x0E,
	DamageTypeChaos:     0x10,
}

type Ailment string

const (
	AilmentBleed   = Ailment("Bleed")
	AilmentPoison  = Ailment("Poison")
	AilmentIgnite  = Ailment("Ignite")
	AilmentChill   = Ailment("Chill")
	AilmentFreeze  = Ailment("Freeze")
	AilmentShock   = Ailment("Shock")
	AilmentScorch  = Ailment("Scorch")
	AilmentBrittle = Ailment("Brittle")
	AilmentSap     = Ailment("Sap")
)

func (Ailment) Values() []Ailment {
	return []Ailment{
		AilmentBleed,
		AilmentPoison,
		AilmentIgnite,
		AilmentChill,
		AilmentFreeze,
		AilmentShock,
		AilmentScorch,
		AilmentBrittle,
		AilmentSap,
	}
}

func (a Ailment) IsNonDamaging() bool {
	_, ok := NonDamagingAilments[a]
	return ok
}

type ElementalAilment string

func (ElementalAilment) Values() []Ailment {
	return []Ailment{
		AilmentIgnite,
		AilmentChill,
		AilmentFreeze,
		AilmentShock,
		AilmentScorch,
		AilmentBrittle,
		AilmentSap,
	}
}

type NonDamagingAilment string

func (NonDamagingAilment) Values() []Ailment {
	return []Ailment{
		AilmentChill,
		AilmentFreeze,
		AilmentShock,
		AilmentScorch,
		AilmentBrittle,
		AilmentSap,
	}
}

type NonDamagingAilmentData struct {
	AssociatedType DamageType
	Alt            bool
	Default        *float64
	Min            float64
	Max            float64
	Precision      float64
	Duration       *float64
}

type SkillType string

const (
	SkillTypeAttack                         = SkillType("Attack")
	SkillTypeSpell                          = SkillType("Spell")
	SkillTypeProjectile                     = SkillType("Projectile")    // Specifically skills which fire projectiles
	SkillTypeDualWieldOnly                  = SkillType("DualWieldOnly") // Attack requires dual wielding only used on Dual Strike
	SkillTypeBuff                           = SkillType("Buff")
	SkillTypeRemoved6                       = SkillType("Removed6")
	SkillTypeMainHandOnly                   = SkillType("MainHandOnly")
	SkillTypeRemoved8                       = SkillType("Removed8")
	SkillTypeMinion                         = SkillType("Minion")
	SkillTypeDamage                         = SkillType("Damage") // Skill hits (not used on attacks because all of them hit)
	SkillTypeArea                           = SkillType("Area")
	SkillTypeDuration                       = SkillType("Duration")
	SkillTypeRequiresShield                 = SkillType("RequiresShield")
	SkillTypeProjectileSpeed                = SkillType("ProjectileSpeed")
	SkillTypeHasReservation                 = SkillType("HasReservation")
	SkillTypeReservationBecomesCost         = SkillType("ReservationBecomesCost")
	SkillTypeTrappable                      = SkillType("Trappable")       // Skill can be turned into a trap
	SkillTypeTotemable                      = SkillType("Totemable")       // Skill can be turned into a totem
	SkillTypeMineable                       = SkillType("Mineable")        // Skill can be turned into a mine
	SkillTypeElementalStatus                = SkillType("ElementalStatus") // Causes elemental status effects but doesn't hit (used on Herald of Ash to allow Elemental Proliferation to apply)
	SkillTypeMinionsCanExplode              = SkillType("MinionsCanExplode")
	SkillTypeRemoved22                      = SkillType("Removed22")
	SkillTypeChains                         = SkillType("Chains")
	SkillTypeMelee                          = SkillType("Melee")
	SkillTypeMeleeSingleTarget              = SkillType("MeleeSingleTarget")
	SkillTypeMulticastable                  = SkillType("Multicastable") // Spell can repeat via Spell Echo
	SkillTypeTotemCastsAlone                = SkillType("TotemCastsAlone")
	SkillTypeMultistrikeable                = SkillType("Multistrikeable") // Attack can repeat via Multistrike
	SkillTypeCausesBurning                  = SkillType("CausesBurning")   // Deals burning damage
	SkillTypeSummonsTotem                   = SkillType("SummonsTotem")
	SkillTypeTotemCastsWhenNotDetached      = SkillType("TotemCastsWhenNotDetached")
	SkillTypeFire                           = SkillType("Fire")
	SkillTypeCold                           = SkillType("Cold")
	SkillTypeLightning                      = SkillType("Lightning")
	SkillTypeTriggerable                    = SkillType("Triggerable")
	SkillTypeTrapped                        = SkillType("Trapped")
	SkillTypeMovement                       = SkillType("Movement")
	SkillTypeRemoved39                      = SkillType("Removed39")
	SkillTypeDamageOverTime                 = SkillType("DamageOverTime")
	SkillTypeRemoteMined                    = SkillType("RemoteMined")
	SkillTypeTriggered                      = SkillType("Triggered")
	SkillTypeVaal                           = SkillType("Vaal")
	SkillTypeAura                           = SkillType("Aura")
	SkillTypeRemoved45                      = SkillType("Removed45")
	SkillTypeCanTargetUnusableCorpse        = SkillType("CanTargetUnusableCorpse") // Doesn't appear to be used at all
	SkillTypeRemoved47                      = SkillType("Removed47")
	SkillTypeRangedAttack                   = SkillType("RangedAttack")
	SkillTypeRemoved49                      = SkillType("Removed49")
	SkillTypeChaos                          = SkillType("Chaos")
	SkillTypeFixedSpeedProjectile           = SkillType("FixedSpeedProjectile") // Not used by any skill
	SkillTypeRemoved52                      = SkillType("Removed52")
	SkillTypeThresholdJewelArea             = SkillType("ThresholdJewelArea") // Allows Burning Arrow and Vigilant Strike to be supported by Inc AoE and Conc Effect
	SkillTypeThresholdJewelProjectile       = SkillType("ThresholdJewelProjectile")
	SkillTypeThresholdJewelDuration         = SkillType("ThresholdJewelDuration") // Allows Burning Arrow to be supported by Inc/Less Duration and Rapid Decay
	SkillTypeThresholdJewelRangedAttack     = SkillType("ThresholdJewelRangedAttack")
	SkillTypeRemoved57                      = SkillType("Removed57")
	SkillTypeChannel                        = SkillType("Channel")
	SkillTypeDegenOnlySpellDamage           = SkillType("DegenOnlySpellDamage") // Allows Contagion Blight and Scorching Ray to be supported by Controlled Destruction
	SkillTypeRemoved60                      = SkillType("Removed60")
	SkillTypeInbuiltTrigger                 = SkillType("InbuiltTrigger") // Skill granted by item that is automatically triggered prevents trigger gems and trap/mine/totem from applying
	SkillTypeGolem                          = SkillType("Golem")
	SkillTypeHerald                         = SkillType("Herald")
	SkillTypeAuraAffectsEnemies             = SkillType("AuraAffectsEnemies") // Used by Death Aura added by Blasphemy
	SkillTypeNoRuthless                     = SkillType("NoRuthless")
	SkillTypeThresholdJewelSpellDamage      = SkillType("ThresholdJewelSpellDamage")
	SkillTypeCascadable                     = SkillType("Cascadable")                     // Spell can cascade via Spell Cascade
	SkillTypeProjectilesFromUser            = SkillType("ProjectilesFromUser")            // Skill can be supported by Volley
	SkillTypeMirageArcherCanUse             = SkillType("MirageArcherCanUse")             // Skill can be supported by Mirage Archer
	SkillTypeProjectileSpiral               = SkillType("ProjectileSpiral")               // Excludes Volley from Vaal Fireball and Vaal Spark
	SkillTypeSingleMainProjectile           = SkillType("SingleMainProjectile")           // Excludes Volley from Spectral Shield Throw
	SkillTypeMinionsPersistWhenSkillRemoved = SkillType("MinionsPersistWhenSkillRemoved") // Excludes Summon Phantasm on Kill from Manifest Dancing Dervish
	SkillTypeProjectileNumber               = SkillType("ProjectileNumber")               // Allows LMP/GMP on Rain of Arrows and Toxic Rain
	SkillTypeWarcry                         = SkillType("Warcry")
	SkillTypeInstant                        = SkillType("Instant") // Instant cast skill
	SkillTypeBrand                          = SkillType("Brand")
	SkillTypeDestroysCorpse                 = SkillType("DestroysCorpse") // Consumes corpses on use
	SkillTypeNonHitChill                    = SkillType("NonHitChill")
	SkillTypeChillingArea                   = SkillType("ChillingArea")
	SkillTypeAppliesCurse                   = SkillType("AppliesCurse")
	SkillTypeCanRapidFire                   = SkillType("CanRapidFire")
	SkillTypeAuraDuration                   = SkillType("AuraDuration")
	SkillTypeAreaSpell                      = SkillType("AreaSpell")
	SkillTypeOR                             = SkillType("OR")
	SkillTypeAND                            = SkillType("AND")
	SkillTypeNOT                            = SkillType("NOT")
	SkillTypePhysical                       = SkillType("Physical")
	SkillTypeAppliesMaim                    = SkillType("AppliesMaim")
	SkillTypeCreatesMinion                  = SkillType("CreatesMinion")
	SkillTypeGuard                          = SkillType("Guard")
	SkillTypeTravel                         = SkillType("Travel")
	SkillTypeBlink                          = SkillType("Blink")
	SkillTypeCanHaveBlessing                = SkillType("CanHaveBlessing")
	SkillTypeProjectilesNotFromUser         = SkillType("ProjectilesNotFromUser")
	SkillTypeAttackInPlaceIsDefault         = SkillType("AttackInPlaceIsDefault")
	SkillTypeNova                           = SkillType("Nova")
	SkillTypeInstantNoRepeatWhenHeld        = SkillType("InstantNoRepeatWhenHeld")
	SkillTypeInstantShiftAttackForLeftMouse = SkillType("InstantShiftAttackForLeftMouse")
	SkillTypeAuraNotOnCaster                = SkillType("AuraNotOnCaster")
	SkillTypeBanner                         = SkillType("Banner")
	SkillTypeRain                           = SkillType("Rain")
	SkillTypeCooldown                       = SkillType("Cooldown")
	SkillTypeThresholdJewelChaining         = SkillType("ThresholdJewelChaining")
	SkillTypeSlam                           = SkillType("Slam")
	SkillTypeStance                         = SkillType("Stance")
	SkillTypeNonRepeatable                  = SkillType("NonRepeatable") // Blood and Sand + Flesh and Stone
	SkillTypeOtherThingUsesSkill            = SkillType("OtherThingUsesSkill")
	SkillTypeSteel                          = SkillType("Steel")
	SkillTypeHex                            = SkillType("Hex")
	SkillTypeMark                           = SkillType("Mark")
	SkillTypeAegis                          = SkillType("Aegis")
	SkillTypeOrb                            = SkillType("Orb")
	SkillTypeKillNoDamageModifiers          = SkillType("KillNoDamageModifiers")
	SkillTypeRandomElement                  = SkillType("RandomElement") // means elements cannot repeat
	SkillTypeLateConsumeCooldown            = SkillType("LateConsumeCooldown")
	SkillTypeArcane                         = SkillType("Arcane") // means it is reliant on amount of mana spent
	SkillTypeFixedCastTime                  = SkillType("FixedCastTime")
	SkillTypeRequiresOffHandNotWeapon       = SkillType("RequiresOffHandNotWeapon")
	SkillTypeLink                           = SkillType("Link")
	SkillTypeBlessing                       = SkillType("Blessing")
	SkillTypeZeroReservation                = SkillType("ZeroReservation")
	//pob:block:src/Data/Global.lua:352
	SkillTypePact = SkillType("Pact")

	SkillTypeRetaliation = SkillType("Retaliation")

	SkillTypeNeverExertable = SkillType("NeverExertable")

	//pob:src/Data/Global.lua:334
	SkillTypeMinionsAreUndamagable = SkillType("MinionsAreUndamagable")
)

func RawToSkillTypes(types []*poe.ActiveSkillType) []SkillType {
	out := make([]SkillType, len(types))
	for i, skillType := range types {
		out[i] = SkillType(skillType.ID)
	}
	return out
}

type WeaponTypeInfo struct {
	OneHand bool
	Melee   bool
	Flag    string
	Label   string
	ModFlag mod.MFlag
}

type ItemClassName string

const (
	None                  ItemClassName = "None"
	Bow                   ItemClassName = "Bow"
	Claw                  ItemClassName = "Claw"
	Dagger                ItemClassName = "Dagger"
	FishingRod            ItemClassName = "Fishing Rod"
	OneHandAxe            ItemClassName = "One Handed Axe"
	OneHandMace           ItemClassName = "One Handed Mace"
	OneHandSword          ItemClassName = "One Handed Sword"
	RuneDagger            ItemClassName = "Rune Dagger"
	Sceptre               ItemClassName = "Sceptre"
	Shield                ItemClassName = "Shield"
	Staff                 ItemClassName = "Staff"
	ThrustingOneHandSword ItemClassName = "Thrusting One Handed Sword"
	TwoHandAxe            ItemClassName = "Two Handed Axe"
	TwoHandMace           ItemClassName = "Two Handed Mace"
	TwoHandSword          ItemClassName = "Two Handed Sword"
	Unarmed               ItemClassName = "Unarmed"
	Wand                  ItemClassName = "Wand"
	Warstaff              ItemClassName = "Warstaff"
)

type JewelRadiusData struct {
	Inner float64
	Outer float64
	Col   string
	Label string
}

var ServerTickRate = 1 / float64(ServerTickTime)
