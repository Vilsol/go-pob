package calculator2

import (
	"fmt"
	"math"
)

type requirementSource struct {
	source        string
	sourceItem    *Item
	sourceGem     *GemInstance
	sourceSlot    string
	Str, Dex, Int float64
}

func (r requirementSource) attr(name string) float64 {
	switch name {
	case "Str":
		return r.Str
	case "Dex":
		return r.Dex
	}
	return r.Int
}

// colorNegative is colorCodes.NEGATIVE.
//
//pob2:src/Data/Global.lua:30
const colorNegative = "^xDD0022"

// luaToString is a number as Lua's tostring and .. print it.
func luaToString(v float64) string {
	return fmt.Sprintf("%.14g", v)
}

// Req<attr>Item is not kept, and a breakdown row has no source name tooltip.
//
//pob2:block:src/Modules/CalcPerform.lua:1896-1992
func processAttributeRequirements(env *Environment, requirementsTable []requirementSource) {
	modDB := env.ModDB
	output := env.Player.Output

	reqMultItem := calcLibMod(modDB, nil, "GlobalAttributeRequirements", "GlobalItemAttributeRequirements")
	reqMultGem := calcLibMod(modDB, nil, "GlobalAttributeRequirements", "GlobalGemAttributeRequirements")
	reqMultWeapon := calcLibMod(modDB, nil, "GlobalAttributeRequirements", "GlobalItemAttributeRequirements", "GlobalWeaponAttributeRequirements")
	output["GlobalItemAttributeRequirements"] = reqMultItem
	output["GlobalGemAttributeRequirements"] = reqMultGem
	output["GlobalWeaponAttributeRequirements"] = reqMultWeapon
	gemAttributeRequirementsSatisfiedByHighestAttribute := modDB.Flag(nil, "GemAttributeRequirementsSatisfiedByHighestAttribute")
	attrTable := []string{"Str", "Dex", "Int"}
	highestAttributeValue := 0.0
	if gemAttributeRequirementsSatisfiedByHighestAttribute {
		// find highest attribute
		for _, attr := range attrTable {
			if checkedAttributeValue := output[attr]; checkedAttributeValue > highestAttributeValue {
				highestAttributeValue = checkedAttributeValue
			}
		}
	}
	ignoreAttrReq := modDB.Flag(nil, "IgnoreAttributeRequirements")
	strengthSatisfiesMeleeFlag := modDB.Flag(nil, "StrengthSatisfiesMeleeWeaponsAndSkills")
	breakdown := env.Player.Breakdown
	for _, attr := range attrTable {
		breakdownAttr := attr
		if breakdown != nil {
			breakdown.setEntry("Req"+attr, &BreakdownEntry{
				Source:  true,
				RowList: []map[string]string{},
				ColList: []BreakdownCol{{attr, "req"}, {"Source", "source"}, {"Source Name", "sourceName"}},
			})
		}
		outVal := 0.0
		for _, reqSource := range requirementsTable {
			if reqSource.attr(attr) > 0 {
				req := 0.0
				switch reqSource.source {
				case "Item":
					if reqSource.sourceItem.Base.Weapon != nil {
						req = math.Floor(reqSource.attr(attr) * reqMultWeapon)
					} else {
						req = math.Floor(reqSource.attr(attr) * reqMultItem)
					}
					output[attr+"RequirementsOn"+reqSource.sourceSlot] = req
				case "Gem":
					req = math.Floor(reqSource.attr(attr) * reqMultGem)
				case "Support Gems":
					req = math.Floor(reqSource.attr(attr))
				}
				// Jarngreipr // if it's a melee weapon or melee skill gem and your Strength is greater than or equal to the Dex/Int requirement
				strengthSatisfiesMelee := strengthSatisfiesMeleeFlag &&
					(reqSource.source == "Item" && reqSource.sourceItem.Base.Weapon != nil && weaponTypeInfo[reqSource.sourceItem.Base.Type].melee ||
						reqSource.source == "Gem" && reqSource.sourceGem != nil && reqSource.sourceGem.GemData != nil && gemTags(reqSource.sourceGem.GemData.Data)["melee"])
				satisfyingAttributeValue := outVal
				if gemAttributeRequirementsSatisfiedByHighestAttribute && reqSource.source == "Gem" {
					satisfyingAttributeValue = highestAttributeValue
				}
				threshold := satisfyingAttributeValue
				if strengthSatisfiesMelee && attr != "Str" {
					threshold = math.Max(satisfyingAttributeValue, output["Str"])
				}
				if req > threshold {
					outVal = req
				}
				if breakdown != nil {
					breakdownAttributeValue := output[breakdownAttr]
					if gemAttributeRequirementsSatisfiedByHighestAttribute && reqSource.source == "Gem" {
						breakdownAttributeValue = highestAttributeValue
					}
					reqThreshold := breakdownAttributeValue
					if strengthSatisfiesMelee && breakdownAttr != "Str" {
						reqThreshold = math.Max(breakdownAttributeValue, output["Str"])
					}
					row := map[string]string{"req": luaToString(req), "reqNum": luaToString(req), "source": reqSource.source}
					if req > reqThreshold {
						row["req"] = colorNegative + luaToString(req)
					}
					switch reqSource.source {
					case "Item":
						item := reqSource.sourceItem
						row["sourceName"] = rarityColors[item.Rarity] + item.Name
					case "Gem":
						gem := reqSource.sourceGem
						row["sourceName"] = sFormat("%s%s ^7%d/%d", gemColor(gem), gem.NameSpec, gem.Level, gem.Quality)
					case "Support Gems":
						attrName := map[string]string{"Str": "Strength", "Dex": "Dexterity", "Int": "Intelligence"}[attr]
						row["sourceName"] = luaToString(reqSource.attr(attr)/5) + " " + attributeColors[attrName] + attrName + " Support Gems"
					}
					e := breakdown.Entries["Req"+breakdownAttr]
					e.RowList = append(e.RowList, row)
				}
			}
		}
		if ignoreAttrReq {
			outVal = 0
		}
		output["Req"+attr+"String"] = 0
		checkedAttributeStr := "Req" + attr
		checkedAttributeValue := output[attr]
		if outVal > output[checkedAttributeStr] {
			output[checkedAttributeStr+"String"] = outVal
			output[checkedAttributeStr] = outVal
			// out.source is the requirement's table, so Lua's out.source == "Gem" never holds
			if env.buildBreakdown() && outVal > checkedAttributeValue {
				env.Player.setOutputString(checkedAttributeStr+"String", colorNegative+luaToString(outVal))
			}
		}
	}
}

// attributeColors is colorCodes[string.upper(attrName)].
//
//pob2:block:src/Data/Global.lua:76-78
var attributeColors = map[string]string{"Strength": "^xE05030", "Dexterity": "^x70FF70", "Intelligence": "^x7070FF"}

// gemColor is gemInstance.color: its granted effect's attribute colour, "^8" without one.
//
//pob2:block:src/Classes/SkillsTab.lua:1351-1361
func gemColor(gem *GemInstance) string {
	grantedEffect := gem.GrantedEffect
	if grantedEffect == nil && gem.GemData != nil {
		grantedEffect = gem.GemData.GrantedEffect
	}
	if grantedEffect == nil {
		return "^8"
	}
	switch grantedEffect.Color {
	case 1:
		return attributeColors["Strength"]
	case 2:
		return attributeColors["Dexterity"]
	case 3:
		return attributeColors["Intelligence"]
	}
	return "^xC8C8C8"
}
