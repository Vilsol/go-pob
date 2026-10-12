
local outPath = arg[1]
assert(outPath, "usage: skill_stat_map_global.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local SRC = "Data/SkillStatMap.lua"

local function makeSkillMod(modName, modType, modVal, flags, keywordFlags, ...)
	return { name = modName, type = modType, value = modVal, flags = flags or 0, keywordFlags = keywordFlags or 0, ... }
end
local function makeFlagMod(modName, ...)
	return makeSkillMod(modName, "FLAG", true, 0, 0, ...)
end
local function makeSkillDataMod(dataKey, dataValue, ...)
	return makeSkillMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
end

local statMap = LoadModule("Data/SkillStatMap", makeSkillMod, makeFlagMod, makeSkillDataMod)

assert(statMap, "REFUSING: LoadModule('Data/SkillStatMap') returned nothing")
assert(next(statMap), "REFUSING: the pinned PoB's global skillStatMap is EMPTY")
assert(data.skillStatMap, "REFUSING: the pinned PoB does not expose data.skillStatMap")

local order, locs, comments, dupes = {}, {}, {}, {}
local seen = {}
local pending = {}
local lineNo = 0
for line in io.lines(SRC) do
	lineNo = lineNo + 1
	local key = line:match('^%["(.-)"%]%s*=')
	if key then
		if seen[key] then
			dupes[#dupes + 1] = string.format("%s (first at :%d, again at :%d)", key, locs[key], lineNo)
		else
			seen[key] = true
			order[#order + 1] = key
		end
		locs[key] = lineNo
		comments[key] = pending
		pending = {}
	elseif line:match("^%-%-") then
		pending[#pending + 1] = line
	elseif not line:match("^\t") then
		pending = {}
	end
end

local scanOnly, runtimeOnly = {}, {}
for key in pairs(seen) do
	if statMap[key] == nil then
		scanOnly[#scanOnly + 1] = key
	end
end
for key in pairs(statMap) do
	if not seen[key] then
		runtimeOnly[#runtimeOnly + 1] = key
	end
end
table.sort(scanOnly)
table.sort(runtimeOnly)
if #scanOnly > 0 or #runtimeOnly > 0 then
	io.stderr:write("REFUSING: the source scan and the loaded table disagree.\n")
	for _, k in ipairs(scanOnly) do io.stderr:write("  scan only:    " .. k .. "\n") end
	for _, k in ipairs(runtimeOnly) do io.stderr:write("  runtime only: " .. k .. "\n") end
	io.stderr:write("The scan supplies declaration order, file:line and PoB's section comments. " ..
		"Emitting anyway would silently drop them for the keys it missed, and nothing downstream " ..
		"would notice. Fix the pattern in this file.\n")
	os.exit(3)
end

local function encodeValue(v)
	local t = type(v)
	if t == "table" then
		local out = { __table = true }
		for k, sub in pairs(v) do
			out[tostring(k)] = encodeValue(sub)
		end
		return out
	elseif t == "boolean" then
		return { __bool = v }
	elseif t == "number" then
		return { __number = v }
	elseif t == "string" then
		return { __string = v }
	end
	return { __other = tostring(v) }
end

local function encodeMod(m)
	local out = {
		name = m.name,
		type = m.type,
		flags = m.flags or 0,
		keywordFlags = m.keywordFlags or 0,
		value = encodeValue(m.value),
		tags = {},
	}
	for _, tag in ipairs(m) do
		table.insert(out.tags, encodeValue(tag))
	end
	local extra = {}
	for k in pairs(m) do
		if type(k) == "string" and k ~= "name" and k ~= "type" and k ~= "value"
			and k ~= "flags" and k ~= "keywordFlags" and k ~= "source" then
			table.insert(extra, k)
		end
	end
	if #extra > 0 then
		table.sort(extra)
		out.extraKeys = extra
	end
	return out
end

local entries = {}
local entryCount, modCount = 0, 0
for _, key in ipairs(order) do
	local entry = statMap[key]
	local mods = {}
	for _, m in ipairs(entry) do
		table.insert(mods, encodeMod(m))
		modCount = modCount + 1
	end
	local scalarKeys = {}
	for k in pairs(entry) do
		if type(k) == "string" then
			table.insert(scalarKeys, k)
		end
	end
	table.sort(scalarKeys)
	local scalars = {}
	for _, k in ipairs(scalarKeys) do
		scalars[k] = encodeValue(entry[k])
	end
	entries[key] = {
		loc = "src/" .. SRC .. ":" .. locs[key],
		comments = comments[key],
		mods = mods,
		scalars = scalars,
	}
	entryCount = entryCount + 1
end

local skillTypeNames = {}
for name, n in pairs(SkillType) do
	skillTypeNames[tostring(n)] = name
end

local function flagBits(t)
	local out = {}
	for name, value in pairs(t) do
		if type(value) == "number" then
			out[name] = value
		end
	end
	return out
end

local fh = assert(io.open(outPath, "w"))
fh:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	skillTypeNames = skillTypeNames,
	modFlagBits = flagBits(ModFlag),
	keywordFlagBits = flagBits(KeywordFlag),
	entryCount = entryCount,
	modCount = modCount,
	duplicateKeys = dupes,
	order = order,
	entries = entries,
}, { indent = true }))
fh:close()

io.stderr:write(string.format("wrote %s (PoB %s, %d entries, %d mods)\n",
	outPath, tostring(launch.versionNumber or "unknown"), entryCount, modCount))
if #dupes > 0 then
	io.stderr:write("note: duplicate keys (Lua last-wins applied): " .. table.concat(dupes, ", ") .. "\n")
end
