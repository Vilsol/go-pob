# Conversion Examples

Some examples that show how to convert code from PoB's Lua to Go

Convenient Lua beautifier: https://goonlinetools.com/lua-beautifier/

---

```lua
if grantedEffect.skillTypes[SkillType.Buff] or grantedEffect.baseFlags.buff then
    preSkillNameList["^" .. skillName:lower() .. " grants "] = {
        addToSkill = {type = "SkillName", skillName = skillName},
        tag = {type = "GlobalEffect", effectType = "Buff"}
    }
    preSkillNameList["^" .. skillName:lower() .. " grants a?n? ?additional "] = {
        addToSkill = {type = "SkillName", skillName = skillName},
        tag = {type = "GlobalEffect", effectType = "Buff"}
    }
end
```

```go
baseFlags, _ := grantedEffect.GetActiveSkill().GetActiveSkillBaseFlagsAndTypes()
if slices.Contains(grantedEffect.GetActiveSkill().ActiveSkillTypes, poe.ActiveSkillTypesByID["Buff"].Key) || baseFlags[poe.SkillFlagBuffs] {
    preSkillNameListCompiled["^"+skillNameLower+" grants "] = CompiledList[modNameListType]{
        Regex: regexp.MustCompile("^" + skillNameLower + " grants "),
        Value: modNameListType{
            addToSkill: mod.SkillName(skillName),
            tag:        mod.GlobalEffect("Buff"),
        },
    }

    preSkillNameListCompiled["^"+skillNameLower+" grants a?n? ?additional "] = CompiledList[modNameListType]{
        Regex: regexp.MustCompile("^" + skillNameLower + " grants a?n? ?additional "),
        Value: modNameListType{
            addToSkill: mod.SkillName(skillName),
            tag:        mod.GlobalEffect("Buff"),
        },
    }
}
```

---

```lua
c["20% chance to Trigger Level 20 Shade Form when you Use a Socketed Skill"] = {
    {
        [1] = {
            flags = 0,
            keywordFlags = 0,
            name = "ExtraSkill",
            type = "LIST",
            value = {level = 20, skillId = "ShadeForm", triggered = true}
        }
    },
    nil
}
```

```go
{
	line: "20% chance to Trigger Level 20 Shade Form when you Use a Socketed Skill",
	mods: []mod.Mod{mod.NewList("ExtraSkill", mod.ExtraSkill{Level: 20, SkillID: "ShadeForm", Triggered: true})},
}
```

---

```lua
c["Zealotry has no Reservation"] = {
    {
        [1] = {
            [1] = {skillId = "SpellDamageAura", type = "SkillId"},
            flags = 0,
            keywordFlags = 0,
            name = "SkillData",
            type = "LIST",
            value = {key = "manaReservationFlat", value = 0}
        },
        [2] = {
            [1] = {skillId = "SpellDamageAura", type = "SkillId"},
            flags = 0,
            keywordFlags = 0,
            name = "SkillData",
            type = "LIST",
            value = {key = "lifeReservationFlat", value = 0}
        },
        [3] = {
            [1] = {skillId = "SpellDamageAura", type = "SkillId"},
            flags = 0,
            keywordFlags = 0,
            name = "SkillData",
            type = "LIST",
            value = {key = "manaReservationPercent", value = 0}
        },
        [4] = {
            [1] = {skillId = "SpellDamageAura", type = "SkillId"},
            flags = 0,
            keywordFlags = 0,
            name = "SkillData",
            type = "LIST",
            value = {key = "lifeReservationPercent", value = 0}
        }
    },
    nil
}
```

```go
{
	line: "Zealotry has no Reservation",
	mods: []mod.Mod{
		mod.NewList("SkillData", mod.SkillData{Key: "manaReservationFlat", Value: 0}).Tag(mod.SkillId("SkillId")),
		mod.NewList("SkillData", mod.SkillData{Key: "lifeReservationFlat", Value: 0}).Tag(mod.SkillId("SkillId")),
		mod.NewList("SkillData", mod.SkillData{Key: "manaReservationPercent", Value: 0}).Tag(mod.SkillId("SkillId")),
		mod.NewList("SkillData", mod.SkillData{Key: "lifeReservationPercent", Value: 0}).Tag(mod.SkillId("SkillId")),
	},
},
```

---

```lua
c["30% increased Effect of Non-Damaging Ailments you inflict during Flask Effect"] = {
    {
        [1] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemyShockEffect",
            type = "INC",
            value = 30
        },
        [2] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemyChillEffect",
            type = "INC",
            value = 30
        },
        [3] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemyFreezeEffect",
            type = "INC",
            value = 30
        },
        [4] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemyScorchEffect",
            type = "INC",
            value = 30
        },
        [5] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemyBrittleEffect",
            type = "INC",
            value = 30
        },
        [6] = {
            [1] = {type = "Condition", var = "UsingFlask"},
            flags = 0,
            keywordFlags = 0,
            name = "EnemySapEffect",
            type = "INC",
            value = 30
        }
    },
    nil
}
```

```go
{
	line: "Zealotry has no Reservation",
	mods: []mod.Mod{
		mod.NewFloat("EnemyShockEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("EnemyChillEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("EnemyFreezeEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("EnemyScorchEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("EnemyBrittleEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
		mod.NewFloat("EnemySapEffect", mod.TypeIncrease, 30).Tag(mod.Condition("UsingFlask")),
	},
},
```