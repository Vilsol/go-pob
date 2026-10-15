
local selfPath = arg[0]
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: item_parse.lua <cases.json> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

local construct = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"))
local cases = assert(json.decode(f:read("*a")))
f:close()

local function modLines(lines)
	local out = {}
	for i, m in ipairs(lines or {}) do
		out[i] = {
			line = m.line,
			extra = m.extra,
			range = m.range,
			crafted = m.crafted or false,
			custom = m.custom or false,
			fractured = m.fractured or false,
			scourge = m.scourge or false,
			variantList = m.variantList,
		}
	end
	return out
end

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
		local item = construct("Item", c.raw)

		local sockets = {}
		for j, s in ipairs(item.sockets or {}) do
			sockets[j] = { color = s.color, group = s.group }
		end

		entry.item = {
			name = item.name,
			title = item.title,
			baseName = item.baseName,
			rarity = item.rarity,
			quality = item.quality,
			itemLevel = item.itemLevel,
			uniqueID = item.uniqueID,
			league = item.league,
			corrupted = item.corrupted or false,
			mirrored = item.mirrored or false,
			fractured = item.fractured or false,
			synthesised = item.synthesised or false,
			shaper = item.shaper or false,
			elder = item.elder or false,
			crusader = item.crusader or false,
			adjudicator = item.adjudicator or false,
			basilisk = item.basilisk or false,
			eyrie = item.eyrie or false,
			cleansing = item.cleansing or false,
			tangle = item.tangle or false,
			evasion = item.evasion,
			armour = item.armour,
			energyShield = item.energyShield,
			checkSection = item.checkSection or false,
			sockets = sockets,
			requirements = numbers(item.requirements),
			implicitModLines = modLines(item.implicitModLines),
			explicitModLines = modLines(item.explicitModLines),
			enchantModLines = modLines(item.enchantModLines),
			scourgeModLines = modLines(item.scourgeModLines),
			classRequirementModLines = modLines(item.classRequirementModLines),
			implicits = item.implicitModLines and #item.implicitModLines or 0,
		}
	end)
	if not ok then
		entry.error = tostring(err)
	end

	results[i] = entry
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. #results .. " items to " .. outPath .. "\n")
