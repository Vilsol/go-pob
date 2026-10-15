
local selfPath = arg[0]
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: item_modlist.lua <cases.json> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

local construct = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"))
local cases = assert(json.decode(f:read("*a")))
f:close()

local function dumpMods(modList)
	local out = {}
	for i, mod in ipairs(modList or {}) do
		local tags = {}
		for _, tag in ipairs(mod) do
			table.insert(tags, tag.type)
		end
		table.sort(tags)

		local value, valueType = nil, type(mod.value)
		if valueType == "number" then
			value = mod.value
		elseif valueType == "boolean" then
			value = mod.value and 1 or 0
		end

		out[i] = {
			name = mod.name,
			type = mod.type,
			value = value,
			valueType = valueType,
			flags = mod.flags or 0,
			keywordFlags = mod.keywordFlags or 0,
			tags = tags,
		}
	end
	return out
end

local results = {}
for i, c in ipairs(cases) do
	local entry = { name = c.name }
	local ok, err = pcall(function()
		local item = construct("Item", c.raw)
		entry.type = item.type
		entry.isWeapon = item.base and item.base.weapon ~= nil or false
		entry.isArmour = item.base and item.base.armour ~= nil or false
		entry.isFlask = item.base and item.base.flask ~= nil or false
		entry.hasBase = item.base ~= nil

		local list = item.modList
		if not list and item.slotModList then
			list = item.slotModList[1]
		end
		entry.mods = dumpMods(list)
		entry.modCount = #(list or {})
	end)
	if not ok then
		entry.error = tostring(err)
	end
	results[i] = entry
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
io.stderr:write("oracle: wrote modlist for " .. #results .. " items to " .. outPath .. "\n")
