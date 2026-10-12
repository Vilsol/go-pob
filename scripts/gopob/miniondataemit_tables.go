package main

var mdSkillDataKeys = map[string]string{
	"cannotBeEvaded": "CannotBeEvaded",
}

var mdModFlags = map[string]string{
	"Ailment":      "mod.MFlagAilment",
	"Area":         "mod.MFlagArea",
	"Attack":       "mod.MFlagAttack",
	"Axe":          "mod.MFlagAxe",
	"Bow":          "mod.MFlagBow",
	"Cast":         "mod.MFlagCast",
	"Claw":         "mod.MFlagClaw",
	"Dagger":       "mod.MFlagDagger",
	"Dot":          "mod.MFlagDot",
	"Fishing":      "mod.MFlagFishing",
	"Hit":          "mod.MFlagHit",
	"Mace":         "mod.MFlagMace",
	"Melee":        "mod.MFlagMelee",
	"MeleeHit":     "mod.MFlagMeleeHit",
	"Projectile":   "mod.MFlagProjectile",
	"SourceMask":   "mod.MFlagSourceMask",
	"Spell":        "mod.MFlagSpell",
	"Staff":        "mod.MFlagStaff",
	"Sword":        "mod.MFlagSword",
	"Unarmed":      "mod.MFlagUnarmed",
	"Wand":         "mod.MFlagWand",
	"Weapon":       "mod.MFlagWeapon",
	"Weapon1H":     "mod.MFlagWeapon1H",
	"Weapon2H":     "mod.MFlagWeapon2H",
	"WeaponMask":   "mod.MFlagWeaponMask",
	"WeaponMelee":  "mod.MFlagWeaponMelee",
	"WeaponRanged": "mod.MFlagWeaponRanged",
}

var mdKeywordFlags = map[string]string{
	"Ailment":      "mod.KeywordFlagAilment",
	"Arrow":        "mod.KeywordFlagArrow",
	"Attack":       "mod.KeywordFlagAttack",
	"Aura":         "mod.KeywordFlagAura",
	"Bleed":        "mod.KeywordFlagBleed",
	"Bow":          "mod.KeywordFlagBow",
	"Brand":        "mod.KeywordFlagBrand",
	"Chaos":        "mod.KeywordFlagChaos",
	"ChaosDot":     "mod.KeywordFlagChaosDot",
	"Cold":         "mod.KeywordFlagCold",
	"ColdDot":      "mod.KeywordFlagColdDot",
	"Curse":        "mod.KeywordFlagCurse",
	"Fire":         "mod.KeywordFlagFire",
	"FireDot":      "mod.KeywordFlagFireDot",
	"Hit":          "mod.KeywordFlagHit",
	"Ignite":       "mod.KeywordFlagIgnite",
	"Lightning":    "mod.KeywordFlagLightning",
	"LightningDot": "mod.KeywordFlagLightningDot",
	"MatchAll":     "mod.KeywordFlagMatchAll",
	"Mine":         "mod.KeywordFlagMine",
	"Minion":       "mod.KeywordFlagMinion",
	"Movement":     "mod.KeywordFlagMovement",
	"Physical":     "mod.KeywordFlagPhysical",
	"PhysicalDot":  "mod.KeywordFlagPhysicalDot",
	"Poison":       "mod.KeywordFlagPoison",
	"Spell":        "mod.KeywordFlagSpell",
	"Totem":        "mod.KeywordFlagTotem",
	"Trap":         "mod.KeywordFlagTrap",
	"Vaal":         "mod.KeywordFlagVaal",
	"Warcry":       "mod.KeywordFlagWarcry",
}

var mdModTypes = map[string]string{
	"BASE":     "mod.TypeBase",
	"INC":      "mod.TypeIncrease",
	"MORE":     "mod.TypeMore",
	"OVERRIDE": "mod.TypeOverride",
	"MAX":      "mod.TypeMAX",
}

var mdNestedListTypes = map[string]string{
	"AllyModifier":   "AllyModifier",
	"MinionModifier": "MinionModifier",
	"PlayerModifier": "PlayerModifier",
}

var mdNumeric = [][2]string{
	{"life", "Life"},
	{"energyShield", "EnergyShield"},
	{"armour", "Armour"},
	{"fireResist", "FireResist"},
	{"coldResist", "ColdResist"},
	{"lightningResist", "LightningResist"},
	{"chaosResist", "ChaosResist"},
	{"damage", "Damage"},
	{"damageSpread", "DamageSpread"},
	{"damageFixup", "DamageFixup"},
	{"attackTime", "AttackTime"},
	{"attackRange", "AttackRange"},
	{"accuracy", "Accuracy"},
	{"evasion", "Evasion"},
}

var mdStrings = [][2]string{
	{"weaponType1", "WeaponType1"},
	{"weaponType2", "WeaponType2"},
	{"limit", "Limit"},
	{"lifeScaling", "LifeScaling"},
}

var mdBools = [][2]string{
	{"baseDamageIgnoresAttackSpeed", "BaseDamageIgnoresAttackSpeed"},
	{"hostile", "Hostile"},
}

const mdMinionInfoStruct = "\n// MinionInfo is one entry of PoB's `data.minions` (src/Modules/Data.lua:1015-1022), the table\n// CalcActiveSkill.lua:592 reads as `env.data.minions[minionType]` when it builds a minion actor.\n//\n// Fields are PoB's own, exported. Absent numeric fields are zero rather than nil: the Lua reads\n// `minionData.armour` / `.energyShield` behind an `if` and `.damageFixup` / `.accuracy` through\n// `or 0` / `or 1`, so zero is the same branch. Accuracy is written out as 1 where the Lua omits\n// it, because `or 1` cannot be expressed by a zero value.\ntype MinionInfo struct {\n\tName            string\n\tLife            float64\n\tEnergyShield    float64\n\tArmour          float64\n\tEvasion         float64\n\tFireResist      float64\n\tColdResist      float64\n\tLightningResist float64\n\tChaosResist     float64\n\tDamage          float64\n\tDamageSpread    float64\n\tDamageFixup     float64\n\tAttackTime      float64\n\tAttackRange     float64\n\tAccuracy        float64\n\tWeaponType1     string\n\tWeaponType2     string\n\tLimit           string\n\t// New in PoB 3.29.\n\tLifeScaling                  string\n\tMonsterTags                  []string\n\tBaseDamageIgnoresAttackSpeed bool\n\tHostile                      bool\n\tSkillList       []string\n\tModList         []mod.Mod\n}\n"

const mdSpectreInit = "\n// src/Modules/Data.lua:1019-1022 -- the spectres are merged into data.minions, so every consumer\n// that resolves a minion type by name reaches them without knowing they came from a second file.\n// The `limit` stamp that loop also applies is already baked into each entry above, because the\n// dumper reads the table AFTER Data.lua has run.\nfunc init() {\n\tfor id, spectre := range Spectres {\n\t\tMinions[id] = spectre\n\t}\n}\n"

type mdTable struct{ goVar, luaFile, doc, trailer string }

var mdTables = map[string]mdTable{
	"minions":  {"Minions", "src/Data/Minions.lua", "// Minions is the whole of src/Data/Minions.lua. It is NOT game data -- go-pob-data\n// ships none of it -- which is why every minion stat read zero here until it was ported.\n//\n// src/Data/Spectres.lua is the other half of the same upstream table and lives in\n// spectre_data.go, whose init() merges it in exactly as src/Modules/Data.lua:1019-1022\n// does.", ""},
	"spectres": {"Spectres", "src/Data/Spectres.lua", "// Spectres is the whole of src/Data/Spectres.lua. Fields are MinionInfo's, the same shape\n// Data/Minions.lua uses -- src/Modules/Data.lua:1015-1022 loads both files into the SAME\n// table -- which is why Limit is spelled out on each entry here and why init() below\n// merges them into Minions rather than keeping a second lookup.\n//\n// The previous revision of this file was a hand transcription of the 13 spectre ids the\n// 3.18 corpus happened to reference, out of PoB's 148.", mdSpectreInit},
}

var mdTableOrder = []string{"minions", "spectres"}
