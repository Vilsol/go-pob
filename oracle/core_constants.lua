
local outPath = arg[1]
assert(outPath, "usage: core_constants.lua <out.json>")

local overlayPath = assert(os.getenv("GAMEDATA_DUMP"),
	"REFUSING: GAMEDATA_DUMP is not set; the game data comes from `gamedatadump core-constants-data`")
local overlay
do
	local f = assert(io.open(overlayPath, "r"), "REFUSING: cannot read " .. overlayPath)
	overlay = assert(require("dkjson").decode(f:read("*a")), "REFUSING: " .. overlayPath .. " is not JSON")
	f:close()
	assert(overlay["Data/Misc.lua"] and overlay["Data/Bosses.lua"],
		"REFUSING: " .. overlayPath .. " carries no Data/Misc.lua or Data/Bosses.lua")
end
local compat = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua")
local pinLoadfile = loadfile
function loadfile(fileName, ...)
	if overlay[fileName] then
		return loadstring(compat.adaptOverlay(fileName, overlay[fileName]), "@" .. fileName)
	end
	return pinLoadfile(fileName, ...)
end

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local missing = {}

local function need(name, value)
	if value == nil then
		table.insert(missing, name)
		return nil
	end
	return value
end

local function exact(v)
	return string.format("%.17g", v)
end

local function numericList(t)
	local values, exacts = {}, {}
	for i = 1, #t do
		assert(type(t[i]) == "number", "REFUSING: non-number at index " .. i)
		values[i] = t[i]
		exacts[i] = exact(t[i])
	end
	return { values = values, exact = exacts }
end

local function sparseNumericMap(t)
	local values, exacts = {}, {}
	for k, v in pairs(t) do
		assert(type(k) == "number", "REFUSING: non-numeric key " .. tostring(k))
		assert(type(v) == "number", "REFUSING: non-number at key " .. tostring(k))
		values[tostring(k)] = v
		exacts[tostring(k)] = exact(v)
	end
	return { values = values, exact = exacts }
end

local function scalarRow(t)
	local out, exacts, dropped = {}, {}, {}
	for k, v in pairs(t) do
		local ty = type(v)
		if ty == "number" then
			out[tostring(k)] = v
			exacts[tostring(k)] = exact(v)
		elseif ty == "string" or ty == "boolean" then
			out[tostring(k)] = v
		else
			dropped[#dropped + 1] = tostring(k)
		end
	end
	table.sort(dropped)
	return { values = out, exact = exacts, dropped = dropped }
end

local tables = {}

for _, name in ipairs({
	"monsterAllyLifeTable",
	"monsterLifeTable",
	"monsterLifeTable2",
	"monsterLifeTable3",
	"monsterEvasionTable",
	"monsterAccuracyTable",
	"monsterDamageTable",
	"monsterAllyDamageTable",
	"monsterArmourTable",
	"monsterPhysConversionMultiTable",
}) do
	local t = need("data." .. name, data[name])
	if t then
		tables[name] = numericList(t)
	end
end

for _, name in ipairs({
	"mapLevelLifeMult",
}) do
	local t = need("data." .. name, data[name])
	if t then
		tables[name] = sparseNumericMap(t)
	end
end

do
	local t = need("data.nonDamagingAilment", data.nonDamagingAilment)
	if t then
		local rows = {}
		for ailment, row in pairs(t) do
			rows[tostring(ailment)] = scalarRow(row)
		end
		tables.nonDamagingAilment = { rows = rows }
	end
end

do
	local t = need("data.keystones", data.keystones)
	if t then
		local names = {}
		for i = 1, #t do
			names[i] = tostring(t[i])
		end
		tables.keystones = { strings = names }
	end
end

do
	local t = need("data.jewelRadius", data.jewelRadius)
	if t then
		local rows = {}
		for i = 1, #t do
			rows[i] = scalarRow(t[i])
		end
		tables.jewelRadius = { rows = rows }
	end
end

do
	local t = need("data.unarmedWeaponData", data.unarmedWeaponData)
	if t then
		local rows, n = {}, 0
		for i = 0, 6 do
			assert(t[i], "REFUSING: data.unarmedWeaponData has no class index " .. i)
			rows[tostring(i)] = scalarRow(t[i])
			n = n + 1
		end
		local total = 0
		for _ in pairs(t) do total = total + 1 end
		assert(total == n, string.format(
			"REFUSING: data.unarmedWeaponData holds %d rows but this dumper reads 7 (indices 0-6). " ..
			"PoB grew or lost a class; model it here rather than dropping it silently.", total))
		tables.unarmedWeaponData = { rows = rows }
	end
end

do
	local t = need("data.weaponTypeInfo", data.weaponTypeInfo)
	if t then
		local rows = {}
		for name, row in pairs(t) do
			rows[tostring(name)] = scalarRow(row)
		end
		tables.weaponTypeInfo = { rows = rows }
	end
end

do
	local t = need("data.bossStats", data.bossStats)
	if t then
		tables.bossStats = scalarRow(t)
	end
end

do
	local t = need("data.misc", data.misc)
	if t then
		tables.misc = scalarRow(t)
	end
end

do
	local t = need("data.characterConstants", data.characterConstants)
	if t then
		tables.characterConstants = scalarRow(t)
	end
end

if #missing > 0 then
	io.stderr:write("REFUSING: the pinned PoB does not expose these declared tables:\n")
	for _, name in ipairs(missing) do
		io.stderr:write("  " .. name .. "\n")
	end
	io.stderr:write("A partial dump would produce a Go file that compiles and is silently missing " ..
		"a table. Either PoB moved them (update this file) or the wrong checkout is loaded.\n")
	os.exit(3)
end

local out = {
	pobVersion = tostring(launch.versionNumber or "unknown"),
	treeVersion = tostring(latestTreeVersion or "unknown"),
	tables = tables,
}

local file = assert(io.open(outPath, "w"),
	"REFUSING: cannot write " .. outPath)
file:write(json.encode(out, { indent = true }))
file:close()

local n = 0
for _ in pairs(tables) do n = n + 1 end
io.stderr:write(string.format("wrote %s (PoB %s, tree %s, %d tables)\n",
	outPath, out.pobVersion, out.treeVersion, n))
