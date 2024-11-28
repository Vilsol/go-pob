package calculator

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

func FLAG(modName string) mod.Mod {
	return MOD(modName, mod.TypeFlag, true)
}

func MOD(modName string, modType mod.Type, modVal any) mod.Mod {
	var realMod mod.Mod
	switch modType {
	case mod.TypeList:
		realMod = mod.NewList(modName, modVal)
	case mod.TypeFlag:
		realMod = mod.NewFlag(modName, modVal.(bool))
	default:
		switch modVal := modVal.(type) {
		case int:
			realMod = mod.NewFloat(modName, modType, float64(modVal))
		case *float64:
			realMod = mod.NewFloat(modName, modType, *modVal)
		default:
			realMod = mod.NewFloat(modName, modType, modVal.(float64))
		}
	}
	return realMod
}

func buildModListForNodeList(env *Environment, nodes map[string]data.Node) *ModList { // TODO ADD: finishJewels)
	// Initialise radius jewels
	/* *
	for _, rad in pairs(env.radiusJewelList) do
		wipeTable(rad.data)
		rad.data.modSource = "Tree:"..rad.nodeId
	end
	/* */

	// Add node modifiers
	var modList = NewModList()
	for _, node := range nodes {
		var nodeModList = buildModListForNode(env, node)
		modList.AddDB(nodeModList)

		// TODO: Is this still a good idea?
		/* *
		if env.Mode == "MAIN" {
			node.finalModList = nodeModList
		}
		/* */
	}

	/* *
	// TODO
	if finishJewels then
		// Process extra radius nodes; these are unallocated nodes near conversion or threshold jewels that need to be processed
		for _, node in pairs(env.extraRadiusNodeList) do
			local nodeModList = calcs.buildModListForNode(env, node)
			if env.mode == "MAIN" then
				node.finalModList = nodeModList
			end
		end

		-- Finalise radius jewels
		for _, rad in pairs(env.radiusJewelList) do
			rad.func(nil, modList, rad.data)
			if env.mode == "MAIN" then
				if not rad.item.jewelRadiusData then
					rad.item.jewelRadiusData = { }
				end
				rad.item.jewelRadiusData[rad.nodeId] = rad.data
			end
		end
	end
	/* */

	return modList
}

func buildModListForNode(env *Environment, node data.Node) *ModList {
	var modList = NewModList()
	for i, stat := range node.Stats {
		var mods, err = parseMod(stat, i)
		if strings.Trim(err, " ") != "" {
			env.DebugErrors = append(env.DebugErrors, "Error parsing Passive Node ("+*node.Name+") mod: "+err+", stat text: "+stat+", with "+strconv.Itoa(len(mods))+" mods found")
		}
		for _, mod := range mods {
			modList.AddMod(mod)
		}
	}

	/* *
	// TODO
	-- Run first pass radius jewels
	for _, rad in pairs(env.radiusJewelList) do
		if rad.type == "Other" and rad.nodes[node.id] and rad.nodes[node.id].type ~= "Mastery" then
			rad.func(node, modList, rad.data)
		end
	end

	if modList:Flag(nil, "PassiveSkillHasNoEffect") or (env.allocNodes[node.id] and modList:Flag(nil, "AllocatedPassiveSkillHasNoEffect")) then
		wipeTable(modList)
	end

	-- Apply effect scaling
	local scale = calcLib.mod(modList, nil, "PassiveSkillEffect")
	if scale ~= 1 then
		local scaledList = new("ModList")
		scaledList:ScaleAddList(modList, scale)
		modList = scaledList
	end

	-- Run second pass radius jewels
	for _, rad in pairs(env.radiusJewelList) do
		if rad.nodes[node.id] and rad.nodes[node.id].type ~= "Mastery" and (rad.type == "Threshold" or (rad.type == "Self" and env.allocNodes[node.id]) or (rad.type == "SelfUnalloc" and not env.allocNodes[node.id])) then
			rad.func(node, modList, rad.data)
		end
	end

	if modList:Flag(nil, "PassiveSkillHasOtherEffect") then
		for i, mod in ipairs(modList:List(skillCfg, "NodeModifier")) do
			if i == 1 then wipeTable(modList) end
			modList:AddMod(mod.mod)
		end
	end

	node.grantedSkills = { }
	for _, skill in ipairs(modList:List(nil, "ExtraSkill")) do
		if skill.name ~= "Unknown" then
			t_insert(node.grantedSkills, {
				skillId = skill.skillId,
				level = skill.level,
				noSupports = true,
				source = "Tree:"..node.id
			})
		end
	end

	if modList:Flag(nil, "CanExplode") then
		t_insert(env.explodeSources, node)
	end
	/* */

	return modList
}
