local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local t_insert, m_max = table.insert, math.max

local function buildMods(displayItem, sourceId)
	local modList, modGroups, groupIndexes = {}, {}, {}
	if sourceId == "EXARCH" or sourceId == "EATER" then
		for i, mod in pairs(displayItem.affixes) do
			if displayItem:GetModSpawnWeight(mod) > 0 and sourceId:lower() == mod.type:lower() then
				local modLabel = table.concat(mod, "/")
				local group = mod.group:gsub("PinnaclePresence", ""):gsub("UniquePresence", "")
				if not groupIndexes[group] then
					t_insert(modList, {})
					t_insert(modGroups, { label = modLabel, mod = mod, modListIndex = #modList, defaultOrder = i })
					groupIndexes[group] = #modGroups
				end
				t_insert(modList[groupIndexes[group]], { label = modLabel, mod = mod, id = i, defaultOrder = i })
			end
		end
		table.sort(modGroups, function(a, b)
			local modA, modB = a.mod, b.mod
			for i = 1, m_max(#modA, #modB) do
				if not modA[i] then return true
				elseif not modB[i] then return false
				elseif modA.statOrder[i] ~= modB.statOrder[i] then return modA.statOrder[i] < modB.statOrder[i] end
			end
			return modA.level > modB.level
		end)
		for i, _ in pairs(modList) do
			table.sort(modList[i], function(a, b)
				local modA, modB = a.mod, b.mod
				if modA.group ~= modB.group then
					if modA.group:match("PinnaclePresence") then return false
					elseif modB.group:match("PinnaclePresence") then return true
					elseif modA.group:match("UniquePresence") then return false
					else return true end
				end
				for j = 1, m_max(#modA, #modB) do
					if not modA[j] then return true
					elseif not modB[j] then return false
					elseif modA.statOrder[j] ~= modB.statOrder[j] then return modA.statOrder[j] < modB.statOrder[j]
					else return tonumber(a.defaultOrder:match("%d+$")) < tonumber(b.defaultOrder:match("%d+$")) end
				end
				return modA.level > modB.level
			end)
		end
		for i, _ in pairs(modGroups) do
			modGroups[i].label = modList[modGroups[i].modListIndex][1].label:gsub("%([%d%.]+%-[%d%.]+%)", "#"):gsub("[%d%.]+", "#")
		end
	elseif sourceId == "DelveImplicit" then
		for i, mod in pairs(displayItem.affixes) do
			if displayItem:GetModSpawnWeight(mod) > 0 and mod.type and sourceId:lower() == mod.type:lower() then
				local modLabel = table.concat(mod, "/")
				if not groupIndexes[mod.group] then
					t_insert(modList, {})
					t_insert(modGroups, { label = modLabel, mod = mod, modListIndex = #modList, defaultOrder = i })
					groupIndexes[mod.group] = #modGroups
				end
				t_insert(modList[groupIndexes[mod.group]], { label = modLabel, mod = mod, id = i, defaultOrder = i })
			end
		end
		for i, _ in pairs(modList) do
			table.sort(modList[i], function(a, b) return a.defaultOrder < b.defaultOrder end)
		end
	end
	local out = {}
	for k, g in ipairs(modGroups) do
		local ids = {}
		for j, e in ipairs(modList[g.modListIndex]) do ids[j] = e.id end
		out[k] = { label = g.label, ids = ids }
	end
	return out
end

local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, raw in ipairs(items) do
	local item = new("Item", raw)
	out[k] = { EXARCH = buildMods(item, "EXARCH"), EATER = buildMods(item, "EATER"), DelveImplicit = buildMods(item, "DelveImplicit") }
end
io.open(outPath, "w"):write(json.encode(out))
