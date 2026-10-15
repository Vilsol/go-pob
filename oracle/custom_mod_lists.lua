local inPath, outPath = arg[1], arg[2]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local t_insert, m_max = table.insert, math.max

local function buildMods(displayItem, sourceId)
	local modList = {}
	if sourceId == "MASTER" then
		local excludeGroups = { }
		for _, modLine in ipairs({ displayItem.prefixes, displayItem.suffixes }) do
			for i = 1, (modLine.limit or (displayItem.affixLimit / 2)) do
				if modLine[i] and modLine[i].modId ~= "None" then
					excludeGroups[displayItem.affixes[modLine[i].modId].group] = true
				end
			end
		end
		for i, craft in ipairs(data.masterMods) do
			if craft.types[displayItem.type] and not excludeGroups[craft.group] then
				t_insert(modList, { label = table.concat(craft, "/") .. " ^8(" .. craft.type .. ")", mod = craft, affixType = craft.type, defaultOrder = i })
			end
		end
		table.sort(modList, function(a, b)
			if a.affixType ~= b.affixType then return a.affixType == "Prefix" and b.affixType == "Suffix" else return a.defaultOrder < b.defaultOrder end
		end)
	elseif sourceId == "ESSENCE" then
		for _, essence in pairs(data.essences) do
			local mod = displayItem.affixes[essence.mods[displayItem.type]]
			if mod then
				t_insert(modList, { label = essence.name .. "   " .. "^8[" .. table.concat(mod, "/") .. "]" .. " (" .. mod.type .. ")", essence = essence })
			end
		end
		table.sort(modList, function(a, b)
			if a.essence.type ~= b.essence.type then return a.essence.type > b.essence.type else return a.essence.tier > b.essence.tier end
		end)
	elseif sourceId == "PREFIX" or sourceId == "SUFFIX" then
		for _, mod in pairs(displayItem.affixes) do
			if mod.type and sourceId:lower() == mod.type:lower() and displayItem:GetModSpawnWeight(mod) > 0 then
				t_insert(modList, { label = mod.affix .. "   ^8[" .. table.concat(mod, "/") .. "]", mod = mod })
			end
		end
		table.sort(modList, function(a, b)
			local modA, modB = a.mod, b.mod
			for i = 1, m_max(#modA, #modB) do
				if not modA[i] then return true
				elseif not modB[i] then return false
				elseif modA.statOrder[i] ~= modB.statOrder[i] then return modA.statOrder[i] < modB.statOrder[i] end
			end
			return modA.level > modB.level
		end)
	elseif sourceId == "DELVE" or sourceId == "INCURSION" then
		local baseCategories = require(sourceId == "DELVE" and "Data.DelveDropOnly" or "Data.IncursionDropOnly")
		local modDb = sourceId == "DELVE" and data.itemMods.Delve or data.itemMods.Explicit
		local base = displayItem.base
		local subTypeName = base.subType and base.type .. ": " .. base.subType
		for modId, entry in pairs(baseCategories) do
			local mod = modDb[modId]
			if mod then
				for _, cat in ipairs(entry.categories) do
					if base.type == cat or subTypeName == cat then
						t_insert(modList, { label = table.concat(mod, "/") .. " (" .. mod.type .. ")", mod = mod, affixType = mod.type, defaultOrder = modId })
						break
					end
				end
			end
		end
		table.sort(modList, function(a, b)
			if a.affixType ~= b.affixType then return a.affixType == "Prefix" and b.affixType == "Suffix" else return a.defaultOrder < b.defaultOrder end
		end)
	else
		local source = ({ VEILED = data.veiledMods, NECROPOLIS = data.necropolisMods, BEASTCRAFT = data.beastCraft, FOSSIL = displayItem.affixes })[sourceId]
		for i, mod in pairs(source) do
			local ok = true
			if sourceId == "VEILED" then ok = displayItem:GetModSpawnWeight(mod) > 0
			elseif sourceId == "NECROPOLIS" then ok = displayItem:GetNecropolisModSpawnWeight(mod) > 0
			elseif sourceId == "FOSSIL" then ok = displayItem:CheckIfModIsDelve(mod) and displayItem:GetModSpawnWeight(mod) > 0 end
			if ok then
				t_insert(modList, { label = table.concat(mod, "/") .. " (" .. mod.type .. ")", mod = mod, affixType = mod.type, defaultOrder = i })
			end
		end
		table.sort(modList, function(a, b)
			if a.affixType ~= b.affixType then return a.affixType == "Prefix" and b.affixType == "Suffix" else return a.defaultOrder < b.defaultOrder end
		end)
	end
	local out = {}
	for k, e in ipairs(modList) do out[k] = e.label end
	return out
end

local function sources(displayItem)
	local sourceList = {}
	if displayItem.type ~= "Tincture" and displayItem.type ~= "Graft" then
		if displayItem.type ~= "Jewel" then t_insert(sourceList, "MASTER") end
		if displayItem.type ~= "Jewel" and displayItem.type ~= "Flask" and displayItem.type ~= "Graft" then
			t_insert(sourceList, "ESSENCE"); t_insert(sourceList, "VEILED"); t_insert(sourceList, "BEASTCRAFT")
		end
		if displayItem.type == "Helmet" or displayItem.type == "Body Armour" or displayItem.type == "Gloves" or displayItem.type == "Boots" then
			t_insert(sourceList, "NECROPOLIS")
		end
		if not displayItem.clusterJewel and displayItem.type ~= "Flask" and displayItem.type ~= "Graft" then
			t_insert(sourceList, "FOSSIL"); t_insert(sourceList, "DELVE"); t_insert(sourceList, "INCURSION")
		end
		if not displayItem.crafted then t_insert(sourceList, "PREFIX"); t_insert(sourceList, "SUFFIX") end
	end
	if displayItem.rareLikeUnique and displayItem.rareLikeUnique.supportsCustomModifiers then
		local newSourceList = {}
		for _, s in ipairs(sourceList) do
			if displayItem.rareLikeUnique.supportsCustomModifiers[s] then t_insert(newSourceList, s) end
		end
		sourceList = newSourceList
	end
	local i = 1
	while sourceList[i] do
		if #buildMods(displayItem, sourceList[i]) == 0 then table.remove(sourceList, i) else i = i + 1 end
	end
	t_insert(sourceList, "CUSTOM")
	return sourceList
end

local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, raw in ipairs(items) do
	local item = construct("Item", raw)
	local res = { sources = sources(item) }
	for _, s in ipairs(res.sources) do
		if s ~= "CUSTOM" then res[s] = buildMods(item, s) end
	end
	out[k] = res
end
io.open(outPath, "w"):write(json.encode(out))
