package mod

type GemProperty struct {
	Key         string
	Value       float64
	KeywordList []string
	Keyword     *string
}

type SkillData struct {
	Key   string
	Value float64
	Merge string
}

type ExtraMinionSkill struct {
	SkillID string
}

type ExtraAuraEffect struct {
	Mod Mod
}

type ExtraAura struct {
	Mod        Mod
	OnlyAllies bool
}

type AffectedByAuraMod struct {
	Mod Mod
}

type MinionModifier struct {
	Mod Mod
}

type ExtraSkillMod struct {
	Mod Mod
}

type EnemyModifier struct {
	Mod Mod
}

type ExtraSkill struct {
	SkillID    string
	SkillName  string
	Level      int
	NoSupports bool
	Triggered  bool
	Source     interface{}
}

type GrantReservedLifeAsAura struct {
	Mod Mod
}

type GrantReservedManaAsAura struct {
	Mod Mod
}

type AffectedByCurseMod struct {
	Mod Mod
}

type WeaponData struct {
	Key   string
	Value float64
}

type ArmourData struct {
	Key   string
	Value float64
}

type ExtraSupport struct {
	SkillID string
	Level   int
}

type ShrineBuff struct {
	Mod Mod
}

type JewelData struct {
	Key   string
	Value any
}

type ConquerorType struct {
	ID   string
	Type string
}

type LegionJewel struct {
	ID        int
	Conqueror ConquerorType
}

type ImpossibleEscapeKeystones struct {
	Key   string
	Value bool
}

type ExtraCurse struct {
	SkillID       string
	SkillName     string
	Level         int
	ApplyToPlayer bool
}

type LinkedSupport struct {
	TargetSlotName string
}

type GrantedAscendancyNode struct {
	Side string
	Name string
}

type ExtraSkillStat struct {
	Key   string
	Value any
}

type SupportedGemProperty struct {
	Keyword string
	Key     string
	Value   int
}
