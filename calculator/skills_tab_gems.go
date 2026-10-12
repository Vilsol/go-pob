package calculator

import (
	"strconv"

	"github.com/Vilsol/go-pob-data/poe"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

//pob:block:src/Classes/SkillsTab.lua:30-58
var DefaultGemLevelModes = []string{"normalMaximum", "corruptedMaximum", "awakenedMaximum", "characterLevel", "levelOne"}

//pob:block:src/Classes/SkillsTab.lua:500-508
func DefaultGemLevelMode(skills pob.Skills) string {
	if skills.MatchGemLevelToCharacterLevel {
		return "characterLevel"
	}
	if skills.DefaultGemLevel != nil {
		if _, err := strconv.ParseFloat(*skills.DefaultGemLevel, 64); err != nil {
			for _, mode := range DefaultGemLevelModes {
				if mode == *skills.DefaultGemLevel {
					return mode
				}
			}
		}
	}
	return "normalMaximum"
}

//pob:src/Classes/SkillsTab.lua:1102-1130
func ProcessGemLevel(build *pob.PathOfBuilding, gemID string, imbued bool) int {
	gemData, ok := raw2.Gems[gemID]
	if !ok {
		return 0
	}
	naturalMaxLevel := gemData.NaturalMaxLevel
	switch mode := DefaultGemLevelMode(build.Skills); {
	case imbued || mode == "levelOne":
		return 1
	case mode == "awakenedMaximum":
		return naturalMaxLevel + 1
	case mode == "corruptedMaximum":
		if raw2.PoBSkills[gemData.GrantedEffectID].PlusVersionOf != "" {
			return naturalMaxLevel
		}
		return naturalMaxLevel + 1
	case mode == "normalMaximum":
		return naturalMaxLevel
	}
	grantedEffect := poe.GrantedEffectByID(gemData.GrantedEffectID)
	if grantedEffect == nil {
		return 1
	}
	levels := raw2.GetCalculatedGrantedEffect(grantedEffect).GetCalculatedLevels()
	maxGemLevel := naturalMaxLevel
	if _, ok := levels[maxGemLevel]; !ok {
		maxGemLevel = 0
		for levels[maxGemLevel+1] != nil {
			maxGemLevel++
		}
	}
	characterLevel := build.Build.Level
	if characterLevel == 0 {
		characterLevel = 1
	}
	for gemLevel := maxGemLevel; gemLevel >= 1; gemLevel-- {
		if level := levels[gemLevel]; level != nil && level.LevelRequirement <= characterLevel {
			return gemLevel
		}
	}
	return 1
}
