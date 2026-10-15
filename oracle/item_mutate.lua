local outPath = arg[1]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local function explicits(item)
	local out = {}
	for _, l in ipairs(item.explicitModLines) do
		if item:CheckModLineVariant(l) then
			out[#out + 1] = (l.mutated and "{mutated}" or "") .. l.line
		end
	end
	return out
end
local result = {}
for name, dbItem in pairs(main.uniqueDB.list) do
	local item = construct("Item", dbItem.raw)
	item:NormaliseQuality()
	local first
	for _, l in ipairs(item.explicitModLines) do
		if l.modId and l.newModId and item:CheckModLineVariant(l) then
			first = l
			break
		end
	end
	if first then
		local entry = { before = explicits(item) }
		local modId, newModId, mutated = first.modId, first.newModId, first.mutated
		item:MutateMod(modId, newModId, not mutated)
		entry.mutated = explicits(item)
		local back
		for _, l in ipairs(item.explicitModLines) do
			if l.modId == newModId and l.newModId == modId and item:CheckModLineVariant(l) then
				back = l
				break
			end
		end
		if back then
			item:MutateMod(back.modId, back.newModId, not back.mutated)
			entry.back = explicits(item)
		end
		result[name] = entry
	end
end
local keys = {}
for k in pairs(result) do
	keys[#keys + 1] = k
end
table.sort(keys)
local f = assert(io.open(outPath, "w"))
f:write(json.encode(result, { indent = true, keyorder = keys }))
f:close()
