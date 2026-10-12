local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local t_insert, m_max = table.insert, math.max

local function controlList(item, affixType, outputTable, outputIndex)
	local extraTags, excludeGroups = {}, {}
	for _, tbl in ipairs({ "prefixes", "suffixes" }) do
		for index = 1, (item[tbl].limit or (item.affixLimit / 2)) do
			if index ~= outputIndex or tbl ~= outputTable then
				local mod = item.affixes[item[tbl][index] and item[tbl][index].modId]
				if mod and mod.group then excludeGroups[mod.group] = true end
			end
		end
	end
	if item.clusterJewel and item.clusterJewelSkill then
		local skill = item.clusterJewel.skills[item.clusterJewelSkill]
		if skill then extraTags[skill.tag] = true end
	end
	local selAffix = item[outputTable][outputIndex] and item[outputTable][outputIndex].modId
	local affixList = {}
	for modId, mod in pairs(item.affixes) do
		if (not affixType or (mod.type == affixType)) and not excludeGroups[mod.group] and not item:CheckIfModIsDelve(mod) then
			if item:CanHaveMod(mod, extraTags) or modId == selAffix then t_insert(affixList, modId) end
		end
	end
	table.sort(affixList, function(a, b)
		local modA, modB = item.affixes[a], item.affixes[b]
		for i = 1, m_max(#modA, #modB) do
			if not modA[i] then return true
			elseif not modB[i] then return false
			elseif modA.statOrder[i] ~= modB.statOrder[i] then return modA.statOrder[i] < modB.statOrder[i] end
		end
		if modA.level ~= modB.level then return modA.level < modB.level end
		return a < b
	end)
	local list, lastSeries = {}, nil
	for _, modId in ipairs(affixList) do
		local mod = item.affixes[modId]
		if not lastSeries or not tableDeepEquals(lastSeries.statOrder, mod.statOrder) then
			lastSeries = { label = table.concat(mod, "/"), modList = {}, statOrder = mod.statOrder }
			t_insert(list, lastSeries)
		end
		t_insert(lastSeries.modList, modId)
		if #lastSeries.modList == 2 then
			lastSeries.label = lastSeries.label:gsub("%(%-?[%d%.]+%-%-?[%d%.]+%)", "#"):gsub("%-?%d+%.?%d*", "#")
		end
	end
	local out = {}
	for k, s in ipairs(list) do out[k] = { label = s.label, ids = s.modList } end
	return out
end

local cases = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, c in ipairs(cases) do
	local item = new("Item", c.raw)
	item.clusterJewelSkill = c.skill
	item.clusterJewelNodeCount = c.count
	wipeTable(item.enchantModLines)
	t_insert(item.enchantModLines, { line = "Adds "..(item.clusterJewelNodeCount or item.clusterJewel.minNodes).." Passive Skills", crafted = true })
	if item.clusterJewel.size == "Large" then
		t_insert(item.enchantModLines, { line = "2 Added Passive Skills are Jewel Sockets", crafted = true })
	elseif item.clusterJewel.size == "Medium" then
		t_insert(item.enchantModLines, { line = "1 Added Passive Skill is a Jewel Socket", crafted = true })
	end
	t_insert(item.enchantModLines, { line = table.concat(item.clusterJewel.skills[item.clusterJewelSkill].enchant, "\n"), crafted = true })
	item:BuildAndParseRaw()
	item:Craft()
	local prefixLimit = item.prefixes.limit or (item.affixLimit / 2)
	local slots = {}
	for i = 1, item.affixLimit do
		if i <= prefixLimit then slots[i] = controlList(item, "Prefix", "prefixes", i)
		else slots[i] = controlList(item, "Suffix", "suffixes", i - prefixLimit) end
	end
	local enchant, explicit = {}, {}
	for i, ml in ipairs(item.enchantModLines) do enchant[i] = ml.line end
	for i, ml in ipairs(item.explicitModLines) do explicit[i] = ml.line end
	out[k] = { enchant = enchant, explicit = explicit, slots = slots }
end
io.open(outPath, "w"):write(json.encode(out))
