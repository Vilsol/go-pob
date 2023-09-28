# Conversion Examples

Some examples that show how to convert code from PoB's Lua to Go

---

```lua
if grantedEffect.skillTypes[SkillType.Buff] or grantedEffect.baseFlags.buff then
  preSkillNameList["^"..skillName:lower().." grants "] = { addToSkill = { type = "SkillName", skillName = skillName }, tag = { type = "GlobalEffect", effectType = "Buff" } }
  preSkillNameList["^"..skillName:lower().." grants a?n? ?additional "] = { addToSkill = { type = "SkillName", skillName = skillName }, tag = { type = "GlobalEffect", effectType = "Buff" } }
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