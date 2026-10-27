package calculator2

import (
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

// Go outputs hold numbers, so Req<attr>String keeps the number where the breakdown colours an unmet one,
// and Req<attr>Item is not kept.
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
	for _, attr := range attrTable {
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
					reqSource.source == "Item" && reqSource.sourceItem.Base.Weapon != nil && weaponTypeInfo[reqSource.sourceItem.Base.Type].melee
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
			}
		}
		if ignoreAttrReq {
			outVal = 0
		}
		output["Req"+attr+"String"] = 0
		checkedAttributeStr := "Req" + attr
		if outVal > output[checkedAttributeStr] {
			output[checkedAttributeStr+"String"] = outVal
			output[checkedAttributeStr] = outVal
		}
	}
}
