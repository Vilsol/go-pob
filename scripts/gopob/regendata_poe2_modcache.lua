-- Dumps PoB-PoE2's shipped src/Data/ModCache.lua, or src/Data/QuestRewards.lua (the quest options
-- ConfigOptions.lua builds from data.questRewards), as JSON for `gopob poe2-modcache-emit`.
-- usage: regendata_poe2_modcache.lua <out.json> <modcache|quests>
-- Each row is {line, mods (null when PoB caches nil), extra (null when nil)}. A mod is
-- {name, type, value, flags, keywordFlags, source (null when nil), tags}; a tag's numeric skillType is
-- written as its SkillType name. An infinite number is written {"$num":"+Inf"} or {"$num":"-Inf"}.
local outPath, which = arg[1], arg[2]
assert(outPath and (which == "modcache" or which == "quests"), "usage: regendata_poe2_modcache.lua <out.json> <modcache|quests>")

dofile("HeadlessWrapper.lua")

local skillTypeName = {}
for name, id in pairs(SkillType) do
	assert(skillTypeName[id] == nil, "REFUSING: two SkillType names share id " .. tostring(id))
	skillTypeName[id] = name
end

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end

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

local encode, encodeMod

local function isMod(t)
	return type(t.name) == "string" and type(t.type) == "string" and t.value ~= nil and t.flags ~= nil
end

function encode(v, path, inTag)
	local ty = type(v)
	if ty == "number" then
		assert(v == v, "REFUSING: NaN at " .. path)
		if v == math.huge or v == -math.huge then
			return v > 0 and '{"$num":"+Inf"}' or '{"$num":"-Inf"}'
		end
		return string.format("%.17g", v)
	elseif ty == "string" then
		return encodeString(v)
	elseif ty == "boolean" then
		return tostring(v)
	elseif ty == "table" then
		if isMod(v) then
			return encodeMod(v, path)
		end
		local n, keyed = 0, false
		for k in pairs(v) do
			n = n + 1
			if type(k) ~= "number" then
				keyed = true
			end
		end
		if n == 0 then
			return "{}"
		end
		local parts = {}
		if not keyed then
			assert(n == #v, "REFUSING: an array with holes at " .. path)
			for i = 1, #v do
				parts[i] = encode(v[i], path .. "[" .. i .. "]", inTag)
			end
			return "[" .. table.concat(parts, ",") .. "]"
		end
		-- A table mixing array and keyed entries keeps its array part under the keys "1", "2", ...
		for _, k in ipairs(sortedKeys(v)) do
			assert(type(k) == "string" or (type(k) == "number" and k >= 1 and k <= #v and k % 1 == 0),
				"REFUSING: a table key that is neither a string nor an array index at " .. path)
			local val = v[k]
			if inTag and k == "skillType" and type(val) == "number" then
				val = assert(skillTypeName[val], "REFUSING: SkillType id " .. val .. " has no name at " .. path)
			elseif inTag and k == "skillTypeList" and type(val) == "table" then
				local names = {}
				for i, id in ipairs(val) do
					names[i] = assert(skillTypeName[id], "REFUSING: SkillType id " .. tostring(id) .. " has no name at " .. path)
				end
				val = names
			end
			parts[#parts + 1] = encodeString(tostring(k)) .. ":" .. encode(val, path .. "." .. k, inTag or type(k) == "number")
		end
		return "{" .. table.concat(parts, ",") .. "}"
	end
	error("REFUSING: a " .. ty .. " at " .. path)
end

function encodeMod(m, path)
	-- EvalMod walks a mod's tags with ipairs, so a tag after a hole is never read; it is not written.
	local tags = {}
	for i, tag in ipairs(m) do
		tags[i] = encode(tag, path .. ".tag" .. i, true)
	end
	for k in pairs(m) do
		if type(k) ~= "number" then
			assert(k == "name" or k == "type" or k == "value" or k == "flags" or k == "keywordFlags" or k == "source",
				"REFUSING: a mod field '" .. tostring(k) .. "' the emitter has no column for at " .. path)
		end
	end
	return "{" .. table.concat({
		'"name":' .. encodeString(m.name),
		'"type":' .. encodeString(m.type),
		'"value":' .. encode(m.value, path .. ".value", false),
		'"flags":' .. encode(m.flags, path .. ".flags", false),
		'"keywordFlags":' .. encode(m.keywordFlags, path .. ".keywordFlags", false),
		'"source":' .. (m.source ~= nil and encode(m.source, path .. ".source", false) or "null"),
		'"tags":[' .. table.concat(tags, ",") .. "]",
	}, ",") .. "}"
end

-- HeadlessWrapper swallows a top-level error's message, so the dump runs under xpcall and says why it refused.
local function dumpQuests()
	local rows = {}
	for i, q in ipairs(data.questRewards) do
		for k in pairs(q) do
			assert(({ Act = 1, Area = 1, AreaLevel = 1, Description = 1, Info = 1, Options = 1, questPoints = 1, Stat = 1, useConfig = 1 })[k],
				"REFUSING: quest field '" .. tostring(k) .. "' the emitter has no column for")
		end
		rows[i] = encode(q, "quest" .. i, false)
	end
	return "[\n" .. table.concat(rows, ",\n") .. "\n]\n"
end

local function main()
if which == "quests" then
	local f = assert(io.open(outPath, "w"))
	f:write(dumpQuests())
	f:close()
	return
end
local cache = dofile("Data/ModCache.lua")
local lines = sortedKeys(cache)
local rows = {}
for i, line in ipairs(lines) do
	local entry = cache[line]
	local mods = "null"
	if entry[1] ~= nil then
		local parts = {}
		for j = 1, #entry[1] do
			assert(isMod(entry[1][j]), "REFUSING: a cached entry that is not a mod at " .. line)
			parts[j] = encodeMod(entry[1][j], line .. "[" .. j .. "]")
		end
		mods = "[" .. table.concat(parts, ",") .. "]"
	end
	local extra = entry[2] ~= nil and encodeString(entry[2]) or "null"
	rows[i] = '{"line":' .. encodeString(line) .. ',"mods":' .. mods .. ',"extra":' .. extra .. "}"
end

local f = assert(io.open(outPath, "w"))
f:write("[\n", table.concat(rows, ",\n"), "\n]\n")
f:close()
end

local ok, err = xpcall(main, debug.traceback)
if not ok then
	io.stderr:write(tostring(err), "\n")
	os.exit(1)
end
