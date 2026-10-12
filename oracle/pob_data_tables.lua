
local outDir = arg[1]
assert(outDir, "usage: pob_data_tables.lua <out-dir>")
outDir = outDir:gsub("/+$", "")

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

local function keysOf(t)
	local out = {}
	for k in pairs(t) do
		out[#out + 1] = tostring(k)
	end
	table.sort(out)
	return out
end

local function scalarsOf(t)
	local out, dropped = {}, {}
	for k, v in pairs(t) do
		local ty = type(v)
		if ty == "number" or ty == "string" or ty == "boolean" then
			out[tostring(k)] = v
		else
			dropped[#dropped + 1] = tostring(k)
		end
	end
	table.sort(dropped)
	if #dropped > 0 then
		out._dropped = dropped
	end
	return out
end

local function flatScalarsOf(t)
	local out, dropped = {}, {}
	for k, v in pairs(t) do
		local ty = type(v)
		if ty == "number" or ty == "string" or ty == "boolean" then
			out[tostring(k)] = v
		elseif ty == "table" and #v > 0 then
			local flat = true
			for i = 1, #v do
				local ety = type(v[i])
				if ety ~= "number" and ety ~= "string" and ety ~= "boolean" then
					flat = false
					break
				end
			end
			if flat then
				for i = 1, #v do
					out[tostring(k) .. "[" .. i .. "]"] = v[i]
				end
			else
				dropped[#dropped + 1] = tostring(k)
			end
		else
			dropped[#dropped + 1] = tostring(k)
		end
	end
	table.sort(dropped)
	if #dropped > 0 then
		out._dropped = dropped
	end
	return out
end

local function exactOf(scalars)
	local out = {}
	for k, v in pairs(scalars) do
		if type(v) == "number" then
			out[k] = string.format("%.17g", v)
		end
	end
	return out
end

local function numericList(t)
	local out = {}
	for i = 1, #t do
		out[i] = t[i]
	end
	return out
end

local tables = {}

for _, name in ipairs({
	"monsterAllyLifeTable",
	"monsterLifeTable",
	"monsterEvasionTable",
	"monsterAccuracyTable",
	"monsterDamageTable",
	"monsterArmourTable",
	"monsterPhysConversionMultiTable",
}) do
	local t = need("data." .. name, data[name])
	if t then
		tables[name] = { kind = "numeric", values = numericList(t) }
	end
end

do
	local t = need("data.mapLevelLifeMult", data.mapLevelLifeMult)
	if t then
		local scalars = scalarsOf(t)
		tables.mapLevelLifeMult = { kind = "scalars", values = scalars, exact = exactOf(scalars) }
	end
end

do
	local t = need("data.nonDamagingAilment", data.nonDamagingAilment)
	if t then
		local rows = {}
		for ailment, row in pairs(t) do
			rows[tostring(ailment)] = scalarsOf(row)
		end
		tables.nonDamagingAilment = { kind = "rows", rows = rows }
	end
end

do
	local t = need("data.misc", data.misc)
	if t then
		local scalars = scalarsOf(t)
		tables.misc = { kind = "scalars", values = scalars, exact = exactOf(scalars) }
	end
end

do
	local t = need("data.weaponTypeInfo", data.weaponTypeInfo)
	if t then
		local rows = {}
		for name, row in pairs(t) do
			rows[tostring(name)] = flatScalarsOf(row)
		end
		tables.weaponTypeInfo = { kind = "rows", rows = rows }
	end
end

do
	local t = need("data.clusterJewels", data.clusterJewels)
	if t then
		local jewels, skills = {}, {}
		for name, jewel in pairs(t.jewels or {}) do
			jewels[tostring(name)] = flatScalarsOf(jewel)
			for skillKey, skill in pairs(jewel.skills or {}) do
				skills[tostring(name) .. "|" .. tostring(skillKey)] = flatScalarsOf(skill)
			end
		end
		tables["clusterJewels.jewels"] = { kind = "rows", rows = jewels }
		tables["clusterJewels.skills"] = { kind = "rows", rows = skills }

		local keystones = {}
		for i = 1, #(t.keystones or {}) do
			keystones[i] = tostring(t.keystones[i])
		end
		table.sort(keystones)
		tables["clusterJewels.keystones"] = { kind = "stringlist", values = keystones }

		local order = scalarsOf(t.notableSortOrder or {})
		tables["clusterJewels.notableSortOrder"] = { kind = "scalars", values = order, exact = exactOf(order) }
	end
end

do
	local t = need("data.keystones", data.keystones)
	if t then
		local names = {}
		for i = 1, #t do
			names[i] = tostring(t[i])
		end
		table.sort(names)
		tables.keystones = { kind = "stringlist", values = names }
	end
end

for _, spec in ipairs({
	{ "pantheons", data.pantheons },
	{ "spectres", data.spectres },
	{ "minions", data.minions },
	{ "skillStatMap", data.skillStatMap },
}) do
	local t = need("data." .. spec[1], spec[2])
	if t then
		tables[spec[1]] = { kind = "keyset", keys = keysOf(t) }
	end
end

do
	local t = need("data.modScalability", data.modScalability)
	if t then
		local rows = {}
		for line, entries in pairs(t) do
			local parts = {}
			for i, entry in ipairs(entries) do
				local one = entry.isScalable and "T" or "F"
				if entry.formats then
					one = one .. ":" .. table.concat(entry.formats, ",")
				end
				parts[i] = one
			end
			rows[tostring(line)] = table.concat(parts, ";")
		end
		tables.modScalability = { kind = "keyvals", values = rows }
	end
end

do
	local t = need("data.itemMods.ItemExclusive", data.itemMods and data.itemMods.ItemExclusive)
	if t then
		local function normaliseModLine(line)
			return line:gsub("%d+%.?%d*", "#")
				:gsub("%(%-?#%-#%)", "#"):lower()
				:gsub("\n", " ")
		end
		local exact, normalised = {}, {}
		for _, mod in pairs(t) do
			for index, line in ipairs(mod) do
				local statOrder = mod.statOrder[index]
				local exactLine = line:lower():gsub("\n", " ")
				local statLine = normaliseModLine(line)
				exact[exactLine] = math.min(exact[exactLine] or math.huge, statOrder)
				normalised[statLine] = math.min(normalised[statLine] or math.huge, statOrder)
			end
		end
		local rows = {}
		local function fmt(v)
			return v and string.format("%.17g", v) or ""
		end
		for k in pairs(exact) do
			rows[k] = true
		end
		for k in pairs(normalised) do
			rows[k] = true
		end
		for k in pairs(rows) do
			rows[k] = fmt(exact[k]) .. "|" .. fmt(normalised[k])
		end
		tables.itemModStatOrder = { kind = "keyvals", values = rows }
	end
end

if #missing > 0 then
	io.stderr:write("REFUSING: the pinned PoB does not expose these declared tables:\n")
	for _, name in ipairs(missing) do
		io.stderr:write("  " .. name .. "\n")
	end
	io.stderr:write("A partial dump would read as 'no drift' for whatever is absent. " ..
		"Either PoB moved them (update this file) or the wrong checkout is loaded.\n")
	os.exit(3)
end

local out = {
	pobVersion = tostring(launch.versionNumber or "unknown"),
	treeVersion = tostring(latestTreeVersion or "unknown"),
	tables = tables,
}

local outPath = outDir .. "/pob-data-tables-" .. out.treeVersion .. ".json"

local file = assert(io.open(outPath, "w"),
	"REFUSING: cannot write " .. outPath .. " -- does the output DIRECTORY exist? " ..
	"This script takes a directory and names the file itself.")
file:write(json.encode(out, { indent = true,
	keyorder = { "pobVersion", "treeVersion", "pobPin", "tables" } }))
file:close()

io.stderr:write(string.format(
	"wrote %s (PoB %s, tree %s, %d tables) -- UNSTAMPED; run oracle/pob_data_tables_stamp.py\n",
	outPath, out.pobVersion, out.treeVersion, (function()
		local n = 0
		for _ in pairs(tables) do n = n + 1 end
		return n
	end)()))
