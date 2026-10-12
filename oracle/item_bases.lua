
local selfPath = arg[0]
local namesPath, outPath = arg[1], arg[2]
assert(namesPath and outPath, "usage: item_bases.lua <names.json> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(namesPath, "r"))
local names = assert(json.decode(f:read("*a")))
f:close()

local function flat(t)
	if type(t) ~= "table" then
		return nil
	end
	local out = {}
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local results = {}
for i, name in ipairs(names) do
	local base = data.itemBases[name]
	local entry = { name = name }

	if not base then
		entry.missing = true
	else
		entry.type = base.type
		entry.subType = base.subType
		entry.socketLimit = base.socketLimit
		entry.hidden = base.hidden or false
		entry.req = flat(base.req) or {}
		entry.armour = flat(base.armour)
		entry.weapon = flat(base.weapon)
		entry.flask = flat(base.flask)
		if base.flask then
			entry.flaskLife = base.flask.life
			entry.flaskMana = base.flask.mana
		end

		local tags = {}
		for tag in pairs(base.tags or {}) do
			table.insert(tags, tag)
		end
		table.sort(tags)
		entry.tags = tags
	end

	results[i] = entry
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. #results .. " item bases to " .. outPath .. "\n")
