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
