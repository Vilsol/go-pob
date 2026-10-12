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
	TypeChance              = Type("CHANCE")
	TypeStatThreshold       = Type("StatThreshold")
	TypeSlotNumber          = Type("SlotNumber")
	TypeSlotName            = Type("SlotName")
	TypeItemCondition       = Type("ItemCondition")
	TypeDistanceRamp        = Type("DistanceRamp")
	TypeMultiplierThreshold = Type("MultiplierThreshold")
	TypeModFlagOr           = Type("ModFlagOr")
	TypeInSlot              = Type("InSlot")
	TypeDummy               = Type("DUMMY")
	TypeGlobalEffect        = Type("GlobalEffect")
	TypeMeleeProximity      = Type("MeleeProximity")
	TypeIgnoreCond          = Type("IgnoreCond")
	TypeSkillPart           = Type("SkillPart")
	TypeLimit               = Type("Limit")
	TypeDisablesItem        = Type("DisablesItem")
	TypeMonsterTag          = Type("MonsterTag")
	TypeDisablerFlag        = Type("Flag")
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
	SourceSkill         = Source("Skill")
)

type KeywordFlag int

func (f *KeywordFlag) Get() KeywordFlag {
	if f != nil {
		return *f
	}
	return 0
}

const (
	KeywordFlagAura         = KeywordFlag(0x00000001)
	KeywordFlagCurse        = KeywordFlag(0x00000002)
	KeywordFlagWarcry       = KeywordFlag(0x00000004)
	KeywordFlagMovement     = KeywordFlag(0x00000008)
	KeywordFlagPhysical     = KeywordFlag(0x00000010)
	KeywordFlagFire         = KeywordFlag(0x00000020)
	KeywordFlagCold         = KeywordFlag(0x00000040)
	KeywordFlagLightning    = KeywordFlag(0x00000080)
	KeywordFlagChaos        = KeywordFlag(0x00000100)
	KeywordFlagVaal         = KeywordFlag(0x00000200)
	KeywordFlagBow          = KeywordFlag(0x00000400)
	KeywordFlagArrow        = KeywordFlag(0x00000800)
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

var KeywordFlagsByName = map[string]KeywordFlag{
	"Aura":         KeywordFlagAura,
	"Curse":        KeywordFlagCurse,
	"Warcry":       KeywordFlagWarcry,
	"Movement":     KeywordFlagMovement,
	"Physical":     KeywordFlagPhysical,
	"Fire":         KeywordFlagFire,
	"Cold":         KeywordFlagCold,
	"Lightning":    KeywordFlagLightning,
	"Chaos":        KeywordFlagChaos,
	"Vaal":         KeywordFlagVaal,
	"Bow":          KeywordFlagBow,
	"Arrow":        KeywordFlagArrow,
	"Trap":         KeywordFlagTrap,
	"Mine":         KeywordFlagMine,
	"Totem":        KeywordFlagTotem,
	"Minion":       KeywordFlagMinion,
	"Attack":       KeywordFlagAttack,
	"Spell":        KeywordFlagSpell,
	"Hit":          KeywordFlagHit,
	"Ailment":      KeywordFlagAilment,
	"Brand":        KeywordFlagBrand,
	"Poison":       KeywordFlagPoison,
	"Bleed":        KeywordFlagBleed,
	"Ignite":       KeywordFlagIgnite,
	"PhysicalDot":  KeywordFlagPhysicalDot,
	"LightningDot": KeywordFlagLightningDot,
	"ColdDot":      KeywordFlagColdDot,
	"FireDot":      KeywordFlagFireDot,
	"ChaosDot":     KeywordFlagChaosDot,
	"MatchAll":     KeywordFlagMatchAll,
}

//pob:src/Data/Global.lua:143-154
func MatchKeywordFlags(keywordFlags KeywordFlag, modKeywordFlags KeywordFlag) bool {
	matchAll := modKeywordFlags&KeywordFlagMatchAll != 0

	modKeywordFlags = modKeywordFlags & MatchAllMask
	keywordFlags = keywordFlags & MatchAllMask

	if matchAll {
		return keywordFlags&modKeywordFlags == modKeywordFlags
	}

	return modKeywordFlags == 0 || keywordFlags&modKeywordFlags != 0
}
