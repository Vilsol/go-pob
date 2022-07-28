package mod

type Tag interface {
	Type() Type
}

type Type string

const (
	TypeIncrease            = Type("INC")
	TypeMore                = Type("MORE")
	TypeBase                = Type("BASE")
	TypeFlag                = Type("FLAG")
	TypeOverride            = Type("OVERRIDE")
	TypeList                = Type("LIST")
	TypeSocketedIn          = Type("SocketedIn")
	TypePerStat             = Type("PerStat")
	TypePercentStat         = Type("PercentStat")
	TypeCondition           = Type("Condition")
	TypeActorCondition      = Type("ActorCondition")
	TypeMultiplier          = Type("Multiplier")
	TypeModFlag             = Type("ModFlag")
	TypeSkillType           = Type("SkillType")
	TypeSkillID             = Type("SkillId")
	TypeGlobal              = Type("Global")
	TypeSkillName           = Type("SkillName")
	TypeMAX                 = Type("MAX")
	TypeStatThreshold       = Type("StatThreshold")
	TypeSlotNumber          = Type("SlotNumber")
	TypeSlotName            = Type("SlotName")
	TypeDistanceRamp        = Type("DistanceRamp")
	TypeMultiplierThreshold = Type("MultiplierThreshold")
	TypeModFlagOr           = Type("ModFlagOr")
	TypeInSlot              = Type("InSlot")
	TypeDummy               = Type("DUMMY")
	TypeGlobalEffect        = Type("GlobalEffect")
	TypeMeleeProximity      = Type("MeleeProximity")
	TypeIgnoreCond          = Type("IgnoreCond")
)

type Source string

const (
	SourceBase          = Source("Base")
	SourceConfig        = Source("Config")
	SourceChill         = Source("Chill")
	SourceBonechill     = Source("Bonechill")
	SourceShock         = Source("Shock")
	SourceScorch        = Source("Scorch")
	SourceBrittle       = Source("Brittle")
	SourceSap           = Source("Sap")
	SourceFeedingFrenzy = Source("Feeding Frenzy")
)

type MFlag int

const (
	FlagAttack       = MFlag(0x00000001)
	FlagSpell        = MFlag(0x00000002)
	FlagHit          = MFlag(0x00000004)
	FlagDot          = MFlag(0x00000008)
	FlagCast         = MFlag(0x00000010)
	FlagMelee        = MFlag(0x00000100)
	FlagArea         = MFlag(0x00000200)
	FlagProjectile   = MFlag(0x00000400)
	FlagSourceMask   = MFlag(0x00000600)
	FlagAilment      = MFlag(0x00000800)
	FlagMeleeHit     = MFlag(0x00001000)
	FlagWeapon       = MFlag(0x00002000)
	FlagAxe          = MFlag(0x00010000)
	FlagBow          = MFlag(0x00020000)
	FlagClaw         = MFlag(0x00040000)
	FlagDagger       = MFlag(0x00080000)
	FlagMace         = MFlag(0x00100000)
	FlagStaff        = MFlag(0x00200000)
	FlagSword        = MFlag(0x00400000)
	FlagWand         = MFlag(0x00800000)
	FlagUnarmed      = MFlag(0x01000000)
	FlagFishing      = MFlag(0x02000000)
	FlagWeaponMelee  = MFlag(0x02000000)
	FlagWeaponRanged = MFlag(0x04000000)
	FlagWeapon1H     = MFlag(0x08000000)
	FlagWeapon2H     = MFlag(0x10000000)
	FlagWeaponMask   = MFlag(0x1FFF0000)
)

type KeywordFlag int

const (
	KeywordFlagAura         = KeywordFlag(0x00000001)
	KeywordFlagCurse        = KeywordFlag(0x00000002)
	KeywordFlagWarcry       = KeywordFlag(0x00000004)
	KeywordFlagMovement     = KeywordFlag(0x00000008)
	KeywordFlagFire         = KeywordFlag(0x00000010)
	KeywordFlagCold         = KeywordFlag(0x00000020)
	KeywordFlagLightning    = KeywordFlag(0x00000040)
	KeywordFlagChaos        = KeywordFlag(0x00000080)
	KeywordFlagVaal         = KeywordFlag(0x00000100)
	KeywordFlagBow          = KeywordFlag(0x00000200)
	KeywordFlagTrap         = KeywordFlag(0x00001000)
	KeywordFlagMine         = KeywordFlag(0x00002000)
	KeywordFlagTotem        = KeywordFlag(0x00004000)
	KeywordFlagMinion       = KeywordFlag(0x00008000)
	KeywordFlagAttack       = KeywordFlag(0x00010000)
	KeywordFlagSpell        = KeywordFlag(0x00020000)
	KeywordFlagHit          = KeywordFlag(0x00040000)
	KeywordFlagAilment      = KeywordFlag(0x00080000)
	KeywordFlagBrand        = KeywordFlag(0x00100000)
	KeywordFlagPoison       = KeywordFlag(0x00200000)
	KeywordFlagBleed        = KeywordFlag(0x00400000)
	KeywordFlagIgnite       = KeywordFlag(0x00800000)
	KeywordFlagPhysicalDot  = KeywordFlag(0x01000000)
	KeywordFlagLightningDot = KeywordFlag(0x02000000)
	KeywordFlagColdDot      = KeywordFlag(0x04000000)
	KeywordFlagFireDot      = KeywordFlag(0x08000000)
	KeywordFlagChaosDot     = KeywordFlag(0x10000000)
	KeywordFlagMatchAll     = KeywordFlag(0x40000000)

	MatchAllMask = ^KeywordFlagMatchAll
)

func MatchKeywordFlags(keywordFlags KeywordFlag, modKeywordFlags KeywordFlag) bool {
	matchAll := modKeywordFlags&KeywordFlagMatchAll != 0
	modKeywordFlags = modKeywordFlags & MatchAllMask
	keywordFlags = keywordFlags & MatchAllMask
	if matchAll {
		return keywordFlags&modKeywordFlags == modKeywordFlags
	}
	return modKeywordFlags == 0 || keywordFlags&modKeywordFlags != 0
}
