local inPath, outPath = arg[1], arg[2]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, raw in ipairs(items) do
	local item = construct("Item", raw)
	local res = {}
	local itemMaxCorruptImplicits = item.corruptImplicitCount or 2
	local specialCorruptLine
	for _, modLine in ipairs(item.implicitModLines) do
		for _, mod in ipairs(modLine.modList or {}) do
			if mod.name == "CorruptImplicitCount" then
				itemMaxCorruptImplicits = itemMaxCorruptImplicits - 1
				specialCorruptLine = modLine
			end
		end
	end
	res.max = itemMaxCorruptImplicits
	local isGlimpse = item.title and item.title:lower():find("glimpse of chaos") and item.isUnique
	local types = { "Corrupted", "ScourgeUpside", "ScourgeDownside" }
	if isGlimpse then
		table.insert(types, "Glimpse of Chaos")
	end
	for _, modType in ipairs(types) do
		local list = {}
		if modType == "Glimpse of Chaos" then
			for modId, mod in pairs(data.itemMods.ItemExclusive) do
				if modId:find("^UniqueSpecialCorruption") then
					table.insert(list, { id = modId, mod = mod })
				end
			end
		end
		for modId, mod in pairs(item.affixes) do
			if mod.type == modType and item:GetModSpawnWeight(mod) > 0 then
				table.insert(list, { id = modId, mod = mod })
			end
		end
		table.sort(list, function(a, b)
			local an = a.mod[1]:lower():gsub("%(.-%)","$"):gsub("[%+%-%%]",""):gsub("%d+","$")
			local bn = b.mod[1]:lower():gsub("%(.-%)","$"):gsub("[%+%-%%]",""):gsub("%d+","$")
			if an ~= bn then return an < bn else return a.mod.level < b.mod.level end
		end)
		local ids = {}
		for i, e in ipairs(list) do ids[i] = e.id end
		res[modType] = ids
		res[modType .. "Mods"] = list
	end
	local source = isGlimpse and "Glimpse of Chaos" or "Corrupted"
	local newItem = construct("Item", item:BuildRaw())
	newItem.corrupted = true
	local newImplicit = {}
	for i = 1, 2 do
		local entry = res[source .. "Mods"][i]
		if entry then
			for _, modLine in ipairs(entry.mod) do
				if entry.mod.modTags[1] then
					table.insert(newImplicit, { line = "{tags:" .. table.concat(entry.mod.modTags, ",") .. "}" .. modLine })
				else
					table.insert(newImplicit, { line = modLine })
				end
			end
		end
	end
	if #newImplicit > 0 then
		wipeTable(newItem.implicitModLines)
		if specialCorruptLine then
			table.insert(newItem.implicitModLines, specialCorruptLine)
		end
		for _, implicit in ipairs(newImplicit) do
			table.insert(newItem.implicitModLines, implicit)
		end
	end
	newItem:BuildAndParseRaw()
	local lines = {}
	for i, modLine in ipairs(newItem.implicitModLines) do lines[i] = modLine.line end
	res.corruptedImplicits = lines
	res.source = source
	for _, modType in ipairs(types) do res[modType .. "Mods"] = nil end
	out[k] = res
end
io.open(outPath, "w"):write(json.encode(out))
