package data

type Gem struct {
	ActiveSkill            *ActiveSkillClass   `json:"active_skill,omitempty"`
	BaseItem               *BaseItem           `json:"base_item"`
	CastTime               *int64              `json:"cast_time,omitempty"`
	IsSupport              bool                `json:"is_support"`
	PerLevel               map[int]PerLevel    `json:"per_level"`
	StatTranslationFile    StatTranslationFile `json:"stat_translation_file"`
	Static                 Static              `json:"static"`
	Tags                   []GemTag            `json:"tags"`
	SecondaryGrantedEffect *string             `json:"secondary_granted_effect,omitempty"`
	SupportGem             *SupportGem         `json:"support_gem,omitempty"`
}

type ActiveSkillClass struct {
	Description              string              `json:"description"`
	DisplayName              string              `json:"display_name"`
	ID                       string              `json:"id"`
	IsManuallyCasted         bool                `json:"is_manually_casted"`
	IsSkillTotem             bool                `json:"is_skill_totem"`
	StatConversions          map[string]string   `json:"stat_conversions"`
	Types                    []string            `json:"types"`
	WeaponRestrictions       []WeaponRestriction `json:"weapon_restrictions"`
	SkillTotemLifeMultiplier *float64            `json:"skill_totem_life_multiplier,omitempty"`
	MinionTypes              []string            `json:"minion_types,omitempty"`
}

type BaseItem struct {
	DisplayName  string       `json:"display_name"`
	ID           string       `json:"id"`
	ReleaseState ReleaseState `json:"release_state"`
}

type PerLevel struct {
	Costs                 *PerLevelCosts        `json:"costs,omitempty"`
	RequiredLevel         *int64                `json:"required_level,omitempty"`
	StatRequirements      *StatRequirements     `json:"stat_requirements,omitempty"`
	Stats                 []*Stat               `json:"stats,omitempty"`
	Reservations          *PerLevelReservations `json:"reservations,omitempty"`
	DamageEffectiveness   *int64                `json:"damage_effectiveness,omitempty"`
	DamageMultiplier      *int64                `json:"damage_multiplier,omitempty"`
	Cooldown              *int64                `json:"cooldown,omitempty"`
	CostMultiplier        *int64                `json:"cost_multiplier,omitempty"`
	AttackSpeedMultiplier *int64                `json:"attack_speed_multiplier,omitempty"`
	StoredUses            *int64                `json:"stored_uses,omitempty"`
}

type PerLevelCosts struct {
	Mana          *int64 `json:"Mana,omitempty"`
	Life          *int64 `json:"Life,omitempty"`
	ManaPerMinute *int64 `json:"ManaPerMinute,omitempty"`
	Es            *int64 `json:"ES,omitempty"`
}

type PerLevelReservations struct {
	ManaFlat    *int64   `json:"mana_flat,omitempty"`
	ManaPercent *float64 `json:"mana_percent,omitempty"`
}

type StatRequirements struct {
	Int *int64 `json:"int,omitempty"`
	Str *int64 `json:"str,omitempty"`
	Dex *int64 `json:"dex,omitempty"`
}

type Stat struct {
	Value *int64  `json:"value,omitempty"`
	ID    *string `json:"id,omitempty"`
}

type Static struct {
	CritChance            *int64              `json:"crit_chance,omitempty"`
	DamageEffectiveness   *int64              `json:"damage_effectiveness,omitempty"`
	QualityStats          []QualityStat       `json:"quality_stats"`
	StatRequirements      *StatRequirements   `json:"stat_requirements,omitempty"`
	Stats                 []*Stat             `json:"stats,omitempty"`
	Cooldown              *int64              `json:"cooldown,omitempty"`
	StoredUses            *int64              `json:"stored_uses,omitempty"`
	AttackSpeedMultiplier *int64              `json:"attack_speed_multiplier,omitempty"`
	Costs                 *StaticCosts        `json:"costs,omitempty"`
	CostMultiplier        *int64              `json:"cost_multiplier,omitempty"`
	Reservations          *StaticReservations `json:"reservations,omitempty"`
	RequiredLevel         *int64              `json:"required_level,omitempty"`
	DamageMultiplier      *int64              `json:"damage_multiplier,omitempty"`
	Vaal                  *VaalClass          `json:"vaal,omitempty"`
	CooldownBypassType    *string             `json:"cooldown_bypass_type,omitempty"`
}

type StaticCosts struct {
	Mana        *int64 `json:"Mana,omitempty"`
	Life        *int64 `json:"Life,omitempty"`
	ManaPercent *int64 `json:"ManaPercent,omitempty"`
}

type QualityStat struct {
	ID    string `json:"id"`
	Set   int64  `json:"set"`
	Value int64  `json:"value"`
}

type StaticReservations struct {
	ManaPercent *float64 `json:"mana_percent,omitempty"`
	LifePercent *float64 `json:"life_percent,omitempty"`
}

type VaalClass struct {
	Souls      int64 `json:"souls"`
	StoredUses int64 `json:"stored_uses"`
}

type SupportGem struct {
	AddedTypes       []string `json:"added_types"`
	AllowedTypes     []string `json:"allowed_types"`
	ExcludedTypes    []string `json:"excluded_types"`
	Letter           string   `json:"letter"`
	SupportsGemsOnly bool     `json:"supports_gems_only"`
}

type WeaponRestriction string

const (
	Bow                   WeaponRestriction = "Bow"
	Claw                  WeaponRestriction = "Claw"
	Dagger                WeaponRestriction = "Dagger"
	FishingRod            WeaponRestriction = "FishingRod"
	OneHandAxe            WeaponRestriction = "One Hand Axe"
	OneHandMace           WeaponRestriction = "One Hand Mace"
	OneHandSword          WeaponRestriction = "One Hand Sword"
	RuneDagger            WeaponRestriction = "Rune Dagger"
	Sceptre               WeaponRestriction = "Sceptre"
	Shield                WeaponRestriction = "Shield"
	Staff                 WeaponRestriction = "Staff"
	ThrustingOneHandSword WeaponRestriction = "Thrusting One Hand Sword"
	TwoHandAxe            WeaponRestriction = "Two Hand Axe"
	TwoHandMace           WeaponRestriction = "Two Hand Mace"
	TwoHandSword          WeaponRestriction = "Two Hand Sword"
	Unarmed               WeaponRestriction = "Unarmed"
	Wand                  WeaponRestriction = "Wand"
	Warstaff              WeaponRestriction = "Warstaff"
)

type ReleaseState string

const (
	GemLegacy     ReleaseState = "legacy"
	GemReleased   ReleaseState = "released"
	GemUnreleased ReleaseState = "unreleased"
)

type StatTranslationFile string

type GemTag string

const (
	GemTagActiveSkill   GemTag = "active_skill"
	GemTagArcane        GemTag = "arcane"
	GemTagArea          GemTag = "area"
	GemTagAttack        GemTag = "attack"
	GemTagAura          GemTag = "aura"
	GemTagBanner        GemTag = "banner"
	GemTagBlessing      GemTag = "blessing"
	GemTagBlink         GemTag = "blink"
	GemTagBrand         GemTag = "brand"
	GemTagChaining      GemTag = "chaining"
	GemTagChannelling   GemTag = "channelling"
	GemTagChaos         GemTag = "chaos"
	GemTagCold          GemTag = "cold"
	GemTagCritical      GemTag = "critical"
	GemTagCurse         GemTag = "curse"
	GemTagDexterity     GemTag = "dexterity"
	GemTagDuration      GemTag = "duration"
	GemTagExceptional   GemTag = "exceptional"
	GemTagFire          GemTag = "fire"
	GemTagGolem         GemTag = "golem"
	GemTagGuard         GemTag = "guard"
	GemTagHerald        GemTag = "herald"
	GemTagHex           GemTag = "hex"
	GemTagIntelligence  GemTag = "intelligence"
	GemTagLightning     GemTag = "lightning"
	GemTagLink          GemTag = "link"
	GemTagLowMaxLevel   GemTag = "low_max_level"
	GemTagMark          GemTag = "mark"
	GemTagMelee         GemTag = "melee"
	GemTagMine          GemTag = "mine"
	GemTagMinion        GemTag = "minion"
	GemTagMovement      GemTag = "movement"
	GemTagNova          GemTag = "nova"
	GemTagORB           GemTag = "orb"
	GemTagPhysical      GemTag = "physical"
	GemTagProjectile    GemTag = "projectile"
	GemTagRandomElement GemTag = "random_element"
	GemTagSlam          GemTag = "slam"
	GemTagSpell         GemTag = "spell"
	GemTagStance        GemTag = "stance"
	GemTagStrength      GemTag = "strength"
	GemTagStrike        GemTag = "strike"
	GemTagSupport       GemTag = "support"
	GemTagTagBow        GemTag = "bow"
	GemTagTotem         GemTag = "totem"
	GemTagTrap          GemTag = "trap"
	GemTagTravel        GemTag = "travel"
	GemTagTrigger       GemTag = "trigger"
	GemTagVaal          GemTag = "vaal"
	GemTagWarcry        GemTag = "warcry"
)
