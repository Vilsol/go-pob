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

local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, entry in ipairs(items) do
	local item = new("Item", entry.raw)
	local prefixLimit = item.prefixes.limit or (item.affixLimit / 2)
	local slots = {}
	for i = 1, item.affixLimit do
		if i <= prefixLimit then
			slots[i] = controlList(item, "Prefix", "prefixes", i)
		else
			slots[i] = controlList(item, "Suffix", "suffixes", i - prefixLimit)
		end
	end
	for _, e in ipairs(entry.edits or {}) do
		local tbl, idx = "prefixes", e.slot
		if e.slot > prefixLimit then tbl, idx = "suffixes", e.slot - prefixLimit end
		item[tbl][idx] = { modId = e.modId, range = e.range }
	end
	item:Craft()
	local lines = {}
	for i, ml in ipairs(item.explicitModLines) do lines[i] = ml.line end
	out[k] = { limit = item.affixLimit, slots = slots, explicit = lines, name = item.name, level = item.requirements.level }
end
io.open(outPath, "w"):write(json.encode(out))
