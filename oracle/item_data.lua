
local selfPath = arg[0]
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: item_data.lua <cases.json> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"))
local cases = assert(json.decode(f:read("*a")))
f:close()

local function numbers(t)
	local out = {}
	if type(t) == "table" then
		for k, v in pairs(t) do
			if type(k) == "string" and type(v) == "number" then
				out[k] = v
			end
		end
	end
	return out
end

local results = {}
for i, c in ipairs(cases) do
	local entry = { name = c.name }
	local ok, err = pcall(function()
		local item = new("Item", c.raw)
		entry.isWeapon = item.base and item.base.weapon ~= nil or false
		entry.isArmour = item.base and item.base.armour ~= nil or false
		entry.isFlask = item.base and item.base.flask ~= nil or false
		if item.weaponData then
			entry.weaponData = numbers(item.weaponData[1])
		end
		entry.armourData = numbers(item.armourData)
		entry.flaskData = numbers(item.flaskData)
	end)
	if not ok then
		entry.error = tostring(err)
	end
	results[i] = entry
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
io.stderr:write("oracle: wrote item data for " .. #results .. " items to " .. outPath .. "\n")
