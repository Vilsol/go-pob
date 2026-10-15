local inPath, outPath = arg[1], arg[2]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
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
	elseif sourceId == "VESTIGIAL" then
		for id, uniqueTitle in pairs(data.vestigialModMappings) do
			local titleLower = uniqueTitle:lower()
			local unique = main.uniqueDB.byTitle[titleLower]
			if not unique
				or (unique.base.type ~= displayItem.type)
				or (displayItem.title:lower() == titleLower) then
				goto vestigialContinue
			end
			local mod = copyTable(data.itemMods.Vestigial[id])
			local modLabel = colorCodes.VESTIGIAL .. table.concat(mod, "/")
			t_insert(modList, { {
				label = modLabel,
				mod = mod,
				type = "vestigial",
				uniqueTitle = uniqueTitle,
				modListIndex = #modList + 1,
				id = id,
			} })
			groupIndexes[mod.group] = #modGroups
			::vestigialContinue::
		end
		table.sort(modList, function(a, b)
			a = a[1]
			b = b[1]
			if a.group == b.group then
				return a.id < b.id
			end
			return a.mod.group < b.mod.group
		end)
		for i, elem in ipairs(modList) do
			local group = elem[1]
			group.modListIndex = i
			table.insert(modGroups, elem[1])
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
	local item = construct("Item", raw)
	local sources = {}
	local titleLower = item.title and item.title:lower()
	local displayDBUnique = main.uniqueDB.byTitle[titleLower]
	if item.rarity == "UNIQUE" and displayDBUnique and data.vestigialUniqueBaseTypes[item.base.type]
		and (not displayDBUnique.source or titleLower == "stasis prison") then
		t_insert(sources, "VESTIGIAL")
	end
	out[k] = { EXARCH = buildMods(item, "EXARCH"), EATER = buildMods(item, "EATER"), sources = sources,
		VESTIGIAL = #sources > 0 and buildMods(item, "VESTIGIAL") or {} }
end
io.open(outPath, "w"):write(json.encode(out))
