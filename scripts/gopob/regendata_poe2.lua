-- Dumps one of PoB-PoE2's loaded data tables as JSON for `gopob poe2-data-emit`.
-- usage: regendata_poe2.lua <out.json> <bases|runes|constants|gems|legion|bosses|boss-skills|runeforging|item-db>
local outPath, which = arg[1], arg[2]
assert(outPath and which, "usage: regendata_poe2.lua <out.json> <bases|runes|constants|gems|legion|bosses|boss-skills|runeforging|item-db>")

-- GAMEDATA_DUMP is `gamedatadump`'s {fileName = source} map: those data files load in place of the pin's.
local overlayPath = os.getenv("GAMEDATA_DUMP")
if overlayPath then
	local f = assert(io.open(overlayPath, "r"), "REFUSING: cannot read " .. overlayPath)
	local overlay = assert(require("dkjson").decode(f:read("*a")), "REFUSING: " .. overlayPath .. " is not JSON")
	f:close()
	local pinLoadfile = loadfile
	function loadfile(fileName, ...)
		if overlay[fileName] then
			return loadstring(overlay[fileName], "@" .. fileName)
		end
		return pinLoadfile(fileName, ...)
	end
	-- Data.lua requires some of its data modules ("Data.VerisiumCrafts")
	for fileName, source in pairs(overlay) do
		local module = fileName:match("^(.+)%.lua$"):gsub("/", ".")
		package.preload[module] = function()
			return assert(loadstring(source, "@" .. fileName))()
		end
	end
end

dofile("HeadlessWrapper.lua")

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys, function(a, b)
		if type(a) == type(b) then
			return a < b
		end
		return type(a) == "number"
	end)
	return keys
end

local function isArray(t)
	local n = 0
	for _ in pairs(t) do
		n = n + 1
	end
	return n == #t
end

local encode
local function encodeString(s)
	return '"' .. s:gsub('[%c"\\]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
function encode(v, path)
	local ty = type(v)
	if ty == "number" then
		assert(v == v and v ~= math.huge and v ~= -math.huge, "REFUSING: non-finite number at " .. path)
		return string.format("%.17g", v)
	elseif ty == "string" then
		return encodeString(v)
	elseif ty == "boolean" then
		return tostring(v)
	elseif ty == "table" then
		local parts = {}
		if next(v) == nil then
			-- a base's `armour = { }` is still an armour base (0 of each defence), unlike a missing one
			if path:sub(-7) == ".armour" then
				return "{}"
			end
			return "null"
		end
		if isArray(v) then
			for i = 1, #v do
				parts[i] = encode(v[i], path .. "[" .. i .. "]")
			end
			return "[" .. table.concat(parts, ",") .. "]"
		end
		for _, k in ipairs(sortedKeys(v)) do
			assert(type(k) == "string", "REFUSING: a table mixing array and keyed entries at " .. path)
			parts[#parts + 1] = encodeString(k) .. ":" .. encode(v[k], path .. "." .. k)
		end
		return "{" .. table.concat(parts, ",") .. "}"
	end
	error("REFUSING: cannot encode a " .. ty .. " at " .. path)
end

local function tagList(set)
	local out = {}
	for tag, on in pairs(set) do
		assert(on == true, "REFUSING: a tag set holds a non-true value")
		out[#out + 1] = tag
	end
	table.sort(out)
	return out
end

local function copy(t, skip)
	local out = {}
	for k, v in pairs(t) do
		if not (skip and skip[k]) then
			out[k] = v
		end
	end
	return out
end

local dumpers = {}

function dumpers.bases()
	local rows = {}
	for _, name in ipairs(sortedKeys(data.itemBases)) do
		local row = copy(data.itemBases[name])
		row.name = name
		row.tags = tagList(row.tags)
		rows[#rows + 1] = row
	end
	return rows
end

function dumpers.runes()
	local rows = {}
	for _, name in ipairs(sortedKeys(data.itemMods.Runes)) do
		local slots = data.itemMods.Runes[name]
		for _, slot in ipairs(sortedKeys(slots)) do
			local mod = slots[slot]
			local row = copy(mod, { tradeHashes = true })
			for i = 1, #mod do
				row[i] = nil
			end
			row.rune, row.slot, row.lines = name, slot, { unpack(mod) }
			if mod.bonded then
				row.bonded = { lines = { unpack(mod.bonded) }, statOrder = mod.bonded.statOrder }
				for k in pairs(mod.bonded) do
					assert(type(k) == "number" or k == "statOrder", "REFUSING: unknown bonded key " .. tostring(k))
				end
			end
			rows[#rows + 1] = row
		end
	end
	return rows
end

function dumpers.constants()
	local out = {}
	for _, name in ipairs({ "misc", "gameConstants", "characterConstants", "monsterConstants", "playerMinionIntrinsicStats",
		"monsterEvasionTable", "monsterAccuracyTable", "monsterLifeTable", "monsterAllyLifeTable", "monsterDamageTable",
		"monsterAllyDamageTable", "monsterArmourTable", "monsterAilmentThresholdTable", "monsterPoiseThresholdTable",
		"minionLevelTable", "hollowPalmAddedPhys" }) do
		out[name] = assert(data[name], "REFUSING: data." .. name .. " is absent")
	end
	-- keyed by skillTotemId, a sparse integer key the encoder refuses
	out.totemLifeMult = {}
	for id, mult in pairs(assert(data.totemLifeMult, "REFUSING: data.totemLifeMult is absent")) do
		out.totemLifeMult[tostring(id)] = mult
	end
	return out
end

function dumpers.gems()
	local rows = {}
	for _, id in ipairs(sortedKeys(data.gems)) do
		local gem = data.gems[id]
		local row = copy(gem, { grantedEffect = true, grantedEffectList = true, additionalGrantedEffects = true })
		row.tags = tagList(row.tags)
		local effect = assert(gem.grantedEffect, "REFUSING: gem " .. id .. " has no grantedEffect")
		row.support = effect.support == true
		row.grantedEffectName = effect.name
		rows[#rows + 1] = row
	end
	return rows
end

function dumpers.legion()
	local legion = LoadModule("Data/TimelessJewelData/LegionPassives")
	local rows = {}
	for i, node in ipairs(legion.nodes) do
		rows[i] = { id = node.id, dn = node.dn, sd = node.sd, ks = node.ks, ["not"] = node["not"], m = node.m }
	end
	return rows
end

function dumpers.bosses()
	local rows = {}
	for _, name in ipairs(sortedKeys(data.bosses)) do
		local row = copy(data.bosses[name])
		row.name = name
		rows[#rows + 1] = row
	end
	return { bosses = rows, bossStats = assert(data.bossStats, "REFUSING: data.bossStats is absent") }
end

dumpers["boss-skills"] = function()
	local rows = {}
	for _, name in ipairs(sortedKeys(data.bossSkills)) do
		local row = copy(data.bossSkills[name])
		row.name = name
		rows[#rows + 1] = row
	end
	return rows
end

function dumpers.runeforging()
	local rows = {}
	for _, name in ipairs(sortedKeys(data.runeforgingCrafts)) do
		local crafts = {}
		for i, craft in ipairs(data.runeforgingCrafts[name]) do
			crafts[i] = { baseTypeId = craft.baseTypeId, name = craft.name }
		end
		rows[#rows + 1] = { base = name, crafts = crafts }
	end
	return rows
end

-- PoB-PoE2's own item databases as Main.lua reads them: every raw unique by its data.uniques key, the generated ones
-- included, and every raw rare template
dumpers["item-db"] = function()
	local uniques = {}
	for _, kind in ipairs(sortedKeys(data.uniques)) do
		local raws = {}
		for i, raw in ipairs(data.uniques[kind]) do
			assert(type(raw) == "string", "REFUSING: data.uniques." .. kind .. "[" .. i .. "] is not a string")
			raws[i] = raw
		end
		uniques[#uniques + 1] = { kind = kind, raws = raws }
	end
	local rares = {}
	for i, raw in ipairs(data.rares) do
		assert(type(raw) == "string", "REFUSING: data.rares[" .. i .. "] is not a string")
		rares[i] = raw
	end
	return { uniques = uniques, rares = rares }
end

local dump = assert(dumpers[which], "REFUSING: no table named " .. which)
local f = assert(io.open(outPath, "w"))
f:write(encode(dump(), which), "\n")
f:close()
