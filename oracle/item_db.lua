
local selfPath = arg[0]
local outPath = arg[1]
assert(outPath, "usage: item_db.lua <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

local construct = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")

local json = require("dkjson")

assert(data.uniques and data.rares, "REFUSING: the pinned PoB exposes no data.uniques / data.rares")

local function graded(item)
	local req = item.requirements or {}
	return {
		name = item.name,
		type = item.type,
		baseName = item.baseName,
		rarity = item.rarity,
		league = item.league or "",
		source = item.source or "",
		level = req.level or 0,
		strMod = req.strMod or 0,
		dexMod = req.dexMod or 0,
		intMod = req.intMod or 0,
		primarySlot = item:GetPrimarySlot() or "",
		variants = item.variantList and #item.variantList or 0,
		explicits = #item.explicitModLines,
	}
end

local uniqueTypes = {}
for itemType in pairs(data.uniques) do
	uniqueTypes[#uniqueTypes + 1] = itemType
end
table.sort(uniqueTypes)

local uniques = {}
for _, itemType in ipairs(uniqueTypes) do
	for i, raw in ipairs(data.uniques[itemType]) do
		assert(type(raw) == "string", "REFUSING: data.uniques." .. itemType .. "[" .. i .. "] is not a string")
		local item = construct("Item", raw, "UNIQUE", true)
		uniques[#uniques + 1] = { kind = itemType, raw = raw, pob = item.base and graded(item) or nil }
	end
end

local rares = {}
for i, raw in ipairs(data.rares) do
	assert(type(raw) == "string", "REFUSING: data.rares[" .. i .. "] is not a string")
	local item = construct("Item", raw, "RARE", true)
	if item.base and item.crafted then
		if item.base.implicit and #item.implicitModLines == 0 then
			local implicitIndex = 1
			for line in item.base.implicit:gmatch("[^\n]+") do
				table.insert(item.implicitModLines, { line = line, modTags = item.base.implicitModTypes and item.base.implicitModTypes[implicitIndex] or { } })
				implicitIndex = implicitIndex + 1
			end
		end
		item:Craft()
	end
	rares[#rares + 1] = { raw = raw, crafted = item:BuildRaw(), pob = item.base and graded(item) or nil }
end

local foulbornMap = {}
local exclusiveMods, foulbornMods = {}, {}
local function lines(mod)
	local out, tags = {}, {}
	for i, line in ipairs(mod or {}) do
		out[i] = line
	end
	for i, tag in ipairs(mod.modTags or {}) do
		tags[i] = tag
	end
	return { lines = out, modTags = tags }
end
for title, map in pairs(data.foulbornMap) do
	foulbornMap[title] = {}
	for origModId, foulModId in pairs(map) do
		foulbornMap[title][origModId] = foulModId
		for _, id in ipairs({ origModId, foulModId }) do
			if data.itemMods.ItemExclusive[id] then
				exclusiveMods[id] = lines(data.itemMods.ItemExclusive[id])
			end
			if data.itemMods.Foulborn[id] then
				foulbornMods[id] = lines(data.itemMods.Foulborn[id])
			end
		end
	end
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable({ pobVersion = tostring(launch.versionNumber or "unknown"), uniques = uniques, rares = rares, foulbornMap = foulbornMap,
	exclusiveMods = exclusiveMods, foulbornMods = foulbornMods }), { indent = true }))
out:close()
io.stderr:write(string.format("item-db: %d uniques, %d rare templates written to %s\n", #uniques, #rares, outPath))
