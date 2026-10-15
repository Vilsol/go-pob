package calculator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

//pob:block:src/Modules/Data.lua:306
var skillColorMap = map[int]string{1: colorStrength, 2: colorDexterity, 3: colorIntelligence, 4: colorNormal}

//pob:block:src/Data/Global.lua:8-12
var rarityColor = map[ItemRarity]string{"NORMAL": colorNormal, "MAGIC": colorMagic, "RARE": "^xFFFF77", "UNIQUE": colorUnique, "RELIC": "^x60C060"}

func (env *Environment) socketGroupSourceLine(source string) (string, bool) {
	switch {
	case strings.HasPrefix(source, "Item:"):
		parts := strings.SplitN(source, ":", 3)
		if len(parts) < 3 {
			return "", false
		}
		color := colorNormal
		if id, err := strconv.Atoi(parts[1]); err == nil {
			if text, ok := itemTextByID(env)[id]; ok {
				if c, ok := rarityColor[ParseItem(text).Rarity]; ok {
					color = c
				}
			}
		}
		return "^7Source: " + color + parts[2], true
	case strings.HasPrefix(source, "Tree:"):
		node := resolveTreeNode(data.TreeVersions[data.LatestTreeVersion].Tree(), strings.TrimPrefix(source, "Tree:"))
		if node.Name == nil {
			return "", false
		}
		return "^7Source: " + colorNormal + *node.Name, true
	}
	return "", false
}

func skillColor(id string) string {
	if c, ok := skillColorMap[raw2.PoBSkills[id].Color]; ok {
		return c
	}
	return colorNormal
}

func skillEffectLine(effect *GemEffect) string {
	color := skillColor(effect.GrantedEffect.ID())
	level, quality := effect.Level, effect.Quality
	levelBonus, qualityBonus := "", ""
	if src := effect.SrcInstance; src != nil {
		level, quality = src.Level, src.Quality
		if effect.Level > src.Level {
			levelBonus = colorMagic + "+" + strconv.Itoa(effect.Level-src.Level) + "^7"
		}
		if effect.Quality > src.Quality {
			qualityBonus = colorMagic + "+" + strconv.Itoa(effect.Quality-src.Quality) + "^7"
		}
	}
	return fmt.Sprintf("%s%s ^7%d%s/%d%s", color, effect.GrantedEffect.Name(), level, levelBonus, quality, qualityBonus)
}

//pob:src/Classes/SkillsTab.lua:1310-1395
//nolint:gocyclo // Mirrors the Lua branch chain.
func (env *Environment) SocketGroupTooltip(group int) []TooltipLine {
	if env == nil || env.Build == nil {
		return nil
	}
	skillSet := activeSkillSetIndex(env.Build.Skills)
	if skillSet < 0 || group < 0 || group >= len(env.Build.Skills.SkillSets[skillSet].Skills) {
		return nil
	}
	socketGroup := &env.Build.Skills.SkillSets[skillSet].Skills[group]
	t := &ItemTooltip{}
	displaySkillList, _ := pob.DisplaySkillListOf(socketGroup).([]*ActiveSkill)

	if socketGroup.Source == "Explode" {
		for _, activeSkill := range displaySkillList {
			if source := activeSkill.ActiveEffect.ExplodeSource; source != nil {
				color, ok := rarityColor[source.Rarity]
				if !ok {
					color = colorNormal
				}
				t.addLine("^7Source: " + color + explodeSourceName(source))
			}
		}
		return t.Lines
	}
	if socketGroup.Enabled && !socketGroup.SlotEnabled {
		t.addLine("^7Note: this group is disabled because it is socketed in the inactive weapon set.")
	}
	if line, ok := env.socketGroupSourceLine(socketGroup.Source); ok {
		t.addLine(line)
		t.addSeparator()
	}

	type displayGem struct{ group, gem int }
	groups := env.Build.Skills.SkillSets[skillSet].Skills
	gemList := make([]displayGem, 0, len(socketGroup.Gems))
	for j := range socketGroup.Gems {
		gemList = append(gemList, displayGem{group, j})
	}
	if socketGroup.SourceAttr != nil && len(displaySkillList) > 0 && socketGroup.Slot != "" && !itemGrantedSkillNoSupports(env, socketGroup) {
		for g := range groups {
			other := &groups[g]
			if other.Slot != socketGroup.Slot || (other.SourceAttr != nil && *other.SourceAttr == *socketGroup.SourceAttr) {
				continue
			}
			for j, gem := range other.Gems {
				if gemData, _ := resolveGemInstance(gem); gemData != nil {
					if row, ok := pobGemForVariant(gemData, gem.SkillID); ok && raw2.PoBSkills[row.GrantedEffectID].Support {
						gemList = append(gemList, displayGem{g, j})
					}
				}
			}
		}
	}
	gemAt := func(d displayGem) pob.Gem { return groups[d.group].Gems[d.gem] }
	gemShown := make([]bool, len(gemList))
	markShown := func(src *pob.Gem) {
		if src == nil {
			return
		}
		for j, d := range gemList {
			if !gemShown[j] && gemAt(d) == *src {
				gemShown[j] = true
				return
			}
		}
	}
	for index, activeSkill := range displaySkillList {
		if index > 0 {
			t.addSeparator()
		}
		t.addLine("^7Active Skill #" + strconv.Itoa(index+1) + ":")
		for _, skillEffect := range activeSkill.EffectList {
			t.addLine(skillEffectLine(skillEffect))
			markShown(skillEffect.SrcInstance)
		}
		if activeSkill.Minion != nil && activeSkill.Minion.Actor != nil && activeSkill.Minion.MainSkill != nil && len(activeSkill.Minion.MainSkill.EffectList) > 0 {
			t.addSeparator()
			t.addLine("^7Active Skill #" + strconv.Itoa(index+1) + "'s Main Minion Skill:")
			activeEffect := activeSkill.Minion.MainSkill.EffectList[0]
			t.addLine(skillEffectLine(activeEffect))
			markShown(activeEffect.SrcInstance)
		}
	}

	showOtherHeader := true
	for j, d := range gemList {
		if gemShown[j] {
			continue
		}
		gemInstance := gemAt(d)
		if showOtherHeader {
			showOtherHeader = false
			t.addSeparator()
			t.addLine("^7Inactive Gems:")
		}
		reason := ""
		gemData, bare := resolveGemInstance(gemInstance)
		row, hasRow := pobGemForVariant(gemData, gemInstance.SkillID)
		if !hasRow && gemData == nil {
			row, hasRow = raw2.Gems[gemInstance.GemID]
		}
		var grantedID string
		switch {
		case hasRow:
			grantedID = row.GrantedEffectID
		case bare != nil:
			grantedID = bare.ID
		}
		display := env.displayEffects[[2]int{d.group, d.gem}]
		color := "^8"
		if grantedID != "" {
			color = skillColor(grantedID)
		}
		switch {
		case grantedID == "":
			reason = "(Unsupported)"
		case !gemInstance.Enabled:
			reason = "(Disabled)"
		case !socketGroup.Enabled || !socketGroup.SlotEnabled:
		case raw2.PoBSkills[grantedID].Support:
			if display != nil && display.Superseded {
				reason = "(Superseded)"
			} else if (display == nil || len(display.IsSupporting) == 0) && len(displaySkillList) > 0 {
				reason = "(Cannot apply to any of the active skills)"
			}
		}
		name := gemInstance.NameSpec
		switch {
		case !hasRow && bare != nil:
			name = (&GrantedEffect{Raw: bare}).Name()
		case hasRow:
			name = pob.SanitiseText(row.Name)
		}
		level, quality := gemInstance.Level, gemInstance.Quality
		levelBonus, qualityBonus := "", ""
		if display != nil {
			if display.Level > gemInstance.Level {
				levelBonus = colorMagic + "+" + strconv.Itoa(display.Level-gemInstance.Level) + "^7"
			}
			if display.Quality > gemInstance.Quality {
				qualityBonus = colorMagic + "+" + strconv.Itoa(display.Quality-gemInstance.Quality) + "^7"
			}
		}
		t.addLine(fmt.Sprintf("%s%s ^7%d%s/%d%s %s", color, name, level, levelBonus, quality, qualityBonus, reason))
	}
	return t.Lines
}
