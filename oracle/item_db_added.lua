local outPath = arg[1]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local result = {}
for _, db in ipairs({ main.uniqueDB, main.rareDB }) do
	for name, item in pairs(db.list) do
		local newItem = construct("Item", item.raw)
		newItem:NormaliseQuality()
		result[name] = { name = newItem.name, explicits = #newItem.explicitModLines, quality = newItem.quality or -1, raw = newItem:BuildRaw() }
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
