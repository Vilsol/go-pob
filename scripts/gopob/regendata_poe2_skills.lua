-- Dumps the part of PoB-PoE2's skill data that has no game source, as JSON for `gopob poe2-skills-emit`:
--   globals  src/Data/Global.lua's SkillType, ModFlag and KeywordFlag enums
--   statmap  src/Data/SkillStatMap.lua, the global stat-to-mod map
--   overlay  the hand-written half of src/Data/Skills/*.lua (from src/Export/Skills/*.txt): per granted
--            effect its directive fields, and per stat set its baseFlags, baseMods and statMap
-- usage: regendata_poe2_skills.lua <out.json> <globals|statmap|overlay>
-- The skill files are loaded again with Data.lua's own mod constructors but without processMod, so a
-- mod carries no source and no notMinionStat tag; the calc applies both when it loads. A mod is
-- {name, type, value, flags, keywordFlags, tags} as the ModCache dump writes it (value and type null
-- when nil); a tag's numeric skillType is written as its SkillType name.
local outPath, which = arg[1], arg[2]
assert(outPath and (which == "globals" or which == "statmap" or which == "overlay"),
	"usage: regendata_poe2_skills.lua <out.json> <globals|statmap|overlay>")

dofile("HeadlessWrapper.lua")

local skillFiles = { "act_str", "act_dex", "act_int", "other", "minion", "spectre", "sup_str", "sup_dex", "sup_int" }

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

local function only(t, allowed, path)
	for k in pairs(t) do
		assert(type(k) == "number" or allowed[k], "REFUSING: a field '" .. tostring(k) .. "' the emitter has no column for at " .. path)
	end
end

local encode, encodeMod

local function isMod(t)
	return type(t.name) == "string" and t.flags ~= nil and t.keywordFlags ~= nil
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
	only(m, { name = true, type = true, value = true, flags = true, keywordFlags = true }, path)
	return "{" .. table.concat({
		'"name":' .. encodeString(m.name),
		'"type":' .. (m.type ~= nil and encode(m.type, path .. ".type", false) or "null"),
		'"value":' .. (m.value ~= nil and encode(m.value, path .. ".value", false) or "null"),
		'"flags":' .. encode(m.flags, path .. ".flags", false),
		'"keywordFlags":' .. encode(m.keywordFlags, path .. ".keywordFlags", false),
		'"tags":[' .. table.concat(tags, ",") .. "]",
	}, ",") .. "}"
end

local function makeSkillMod(modName, modType, modVal, flags, keywordFlags, ...)
	return {
		name = modName,
		type = modType,
		value = modVal,
		flags = flags or 0,
		keywordFlags = keywordFlags or 0,
		...
	}
end
local function makeFlagMod(modName, ...)
	return makeSkillMod(modName, "FLAG", true, 0, 0, ...)
end
local function makeSkillDataMod(dataKey, dataValue, ...)
	return makeSkillMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
end

local function num(v)
	return v ~= nil and encode(v, "scalar", false) or "null"
end

-- A statMap entry is { modOrGroup..., value, mult, div, base, skillFlag }; a group is { mod..., div }.
local function encodeEntry(stat, entry, path)
	only(entry, { value = true, mult = true, div = true, base = true, skillFlag = true }, path)
	local items = {}
	for i, modOrGroup in ipairs(entry) do
		local p = path .. "[" .. i .. "]"
		if modOrGroup.name then
			items[i] = encodeMod(modOrGroup, p)
		else
			only(modOrGroup, { value = true, mult = true, div = true, base = true }, p)
			local mods = {}
			for j, m in ipairs(modOrGroup) do
				assert(isMod(m), "REFUSING: a group member that is not a mod at " .. p)
				mods[j] = encodeMod(m, p .. "[" .. j .. "]")
			end
			items[i] = '{"group":[' .. table.concat(mods, ",") .. '],"value":' .. num(modOrGroup.value) .. ',"mult":' .. num(modOrGroup.mult) ..
				',"div":' .. num(modOrGroup.div) .. ',"base":' .. num(modOrGroup.base) .. "}"
		end
	end
	for k in pairs(entry) do
		assert(type(k) ~= "number" or (k >= 1 and k <= #entry), "REFUSING: a statMap entry with a hole at " .. path)
	end
	return '{"stat":' .. encodeString(stat) .. ',"items":[' .. table.concat(items, ",") .. '],"value":' .. num(entry.value) ..
		',"mult":' .. num(entry.mult) .. ',"div":' .. num(entry.div) .. ',"base":' .. num(entry.base) ..
		',"skillFlag":' .. (entry.skillFlag ~= nil and encodeString(entry.skillFlag) or "null") .. "}"
end

local function encodeStatMap(map, path)
	local rows = {}
	for _, stat in ipairs(sortedKeys(map)) do
		assert(type(stat) == "string", "REFUSING: a statMap key that is not a string at " .. path)
		rows[#rows + 1] = encodeEntry(stat, map[stat], path .. "." .. stat)
	end
	return "[" .. table.concat(rows, ",\n") .. "]"
end

local function encodeEnum(enum, what)
	local rows = {}
	for _, name in ipairs(sortedKeys(enum)) do
		local v = enum[name]
		assert(type(v) == "number" and v >= 0 and v % 1 == 0 and v < 2 ^ 53, "REFUSING: " .. what .. "." .. name .. " is not an exact non-negative integer")
		rows[#rows + 1] = '{"name":' .. encodeString(name) .. ',"value":' .. string.format("%.17g", v) .. "}"
	end
	return "[" .. table.concat(rows, ",") .. "]"
end

local function strList(list, path)
	local parts = {}
	for i, s in ipairs(list) do
		assert(type(s) == "string", "REFUSING: a non-string at " .. path)
		parts[i] = encodeString(s)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local function trueSet(set, path)
	local keys = {}
	for _, k in ipairs(sortedKeys(set)) do
		assert(type(k) == "string" and set[k] == true, "REFUSING: a flag set holds something other than name = true at " .. path)
		keys[#keys + 1] = k
	end
	return strList(keys, path)
end

-- The directives Export/Scripts/skills.lua folds into game fields: #addSkillTypes (into skillTypes),
-- #skill <id> <display name>, and #set <GrantedEffectStatSets id>, which names each stat set the game
-- rows it was exported from. Read from the templates the skill files were generated from.
local function templateDirectives()
	local out = {}
	for _, file in ipairs(skillFiles) do
		local pending, cur
		for line in io.lines("Export/Skills/" .. file .. ".txt") do
			local types = line:match("^#addSkillTypes (.*)$")
			if types then
				pending = {}
				for t in types:gmatch("%a+") do
					pending[#pending + 1] = t
				end
			end
			local args = line:match("^#skill (.*)$")
			if args then
				local id, display = args:match("^(%w+) (.+)$")
				id = id or args
				-- a template that repeats a #skill writes skills[id] twice, and the later assignment wins
				cur = { addSkillTypes = pending, displayName = display, sets = {} }
				out[id] = cur
				pending = nil
			end
			local set = line:match("^#set (%S+)")
			if set then
				assert(cur, "REFUSING: #set " .. set .. " before any #skill in " .. file)
				cur.sets[#cur.sets + 1] = set
			end
		end
	end
	return out
end

local skillFields = {
	-- game-sourced: read from go-pob-data by the calc, not emitted
	name = false, baseTypeName = false, icon = false, color = false, description = false, skillTypes = false,
	castTime = false, qualityStats = false, altQualityStats = false, levels = false, support = false,
	requireSkillTypes = false, addSkillTypes = false, excludeSkillTypes = false, gemFamily = false, isLineage = false,
	flavourText = false, isTrigger = false, ignoreMinionTypes = false, weaponTypes = false, minionSkillTypes = false,
	skillTotemId = false, cannotBeSupported = false, supportGemsOnly = false, statSets = false,
	-- PoB's own
	hidden = true, fromItem = true, fromTree = true, minionList = true, hideFromSideBar = true, addFlags = true,
	grantsThornsDamage = true, legacy = true, minionHasItemSet = true, minionUses = true, preDamageFunc = true,
}
local setFields = {
	label = false, baseEffectiveness = false, incrementalEffectiveness = false, damageIncrementalEffectiveness = false,
	statDescriptionScope = false, constantStats = false, stats = false, notMinionStat = false, levels = false,
	baseFlags = true, baseMods = true, statMap = true,
}

local function flag(name, v, path)
	if v == nil then
		return nil
	end
	assert(v == true, "REFUSING: " .. path .. "." .. name .. " is not true")
	return '"' .. name .. '":true'
end

local function dumpOverlay()
	local skills = {}
	for _, file in ipairs(skillFiles) do
		LoadModule("Data/Skills/" .. file)(skills, makeSkillMod, makeFlagMod, makeSkillDataMod)
	end
	for id in pairs(data.skills) do
		assert(skills[id], "REFUSING: data.skills has " .. id .. " but the skill files do not")
	end
	for id in pairs(skills) do
		assert(data.skills[id], "REFUSING: the skill files have " .. id .. " but data.skills does not")
	end
	local directives = templateDirectives()
	local rows = {}
	for _, id in ipairs(sortedKeys(skills)) do
		local ge, path = skills[id], id
		for k in pairs(ge) do
			assert(skillFields[k] ~= nil, "REFUSING: granted effect field '" .. tostring(k) .. "' is in neither the game nor the overlay list at " .. path)
		end
		local parts = { '"id":' .. encodeString(id) }
		local function add(s)
			if s then
				parts[#parts + 1] = s
			end
		end
		for _, k in ipairs({ "hidden", "fromItem", "fromTree", "hideFromSideBar", "grantsThornsDamage", "legacy", "minionHasItemSet" }) do
			add(flag(k, ge[k], path))
		end
		if ge.minionList then
			add('"minionList":' .. strList(ge.minionList, path .. ".minionList"))
		end
		if ge.addFlags then
			add('"addFlags":' .. trueSet(ge.addFlags, path .. ".addFlags"))
		end
		if ge.minionUses then
			add('"minionUses":' .. trueSet(ge.minionUses, path .. ".minionUses"))
		end
		if ge.preDamageFunc then
			assert(type(ge.preDamageFunc) == "function", "REFUSING: preDamageFunc is not a function at " .. path)
			add('"preDamageFunc":true')
		end
		local d = directives[id]
		if d and d.addSkillTypes then
			for _, t in ipairs(d.addSkillTypes) do
				assert(SkillType[t] and ge.skillTypes[SkillType[t]], "REFUSING: #addSkillTypes " .. t .. " is not in " .. id .. "'s skillTypes")
			end
			add('"addSkillTypes":' .. strList(d.addSkillTypes, path .. ".addSkillTypes"))
		end
		if d and d.displayName then
			add('"displayName":' .. encodeString(d.displayName))
		end
		-- a skill written straight into the skill files has no template, and its sets no game id
		assert(not d or #d.sets == #(ge.statSets or {}), "REFUSING: " .. id .. "'s template has " .. (d and #d.sets or 0) .. " #set lines for " .. #(ge.statSets or {}) .. " stat sets")
		if not d then
			add('"noTemplate":true')
		end
		local sets = {}
		for i, set in ipairs(ge.statSets or {}) do
			local sp = path .. ".statSets[" .. i .. "]"
			for k in pairs(set) do
				assert(setFields[k] ~= nil, "REFUSING: stat set field '" .. tostring(k) .. "' is in neither the game nor the overlay list at " .. sp)
			end
			local s = {}
			if d then
				s[#s + 1] = '"id":' .. encodeString(d.sets[i])
			end
			if set.baseFlags then
				s[#s + 1] = '"baseFlags":' .. trueSet(set.baseFlags, sp .. ".baseFlags")
			end
			if set.baseMods then
				local mods = {}
				for j, m in ipairs(set.baseMods) do
					assert(isMod(m), "REFUSING: a baseMods entry that is not a mod at " .. sp)
					mods[j] = encodeMod(m, sp .. ".baseMods[" .. j .. "]")
				end
				s[#s + 1] = '"baseMods":[' .. table.concat(mods, ",") .. "]"
			end
			if set.statMap then
				s[#s + 1] = '"statMap":' .. encodeStatMap(set.statMap, sp .. ".statMap")
			end
			sets[i] = "{" .. table.concat(s, ",") .. "}"
		end
		if ge.statSets then
			add('"statSets":[' .. table.concat(sets, ",") .. "]")
		end
		rows[#rows + 1] = "{" .. table.concat(parts, ",") .. "}"
	end
	return "[\n" .. table.concat(rows, ",\n") .. "\n]\n"
end

local function main()
	local body
	if which == "globals" then
		body = '{"skillType":' .. encodeEnum(SkillType, "SkillType") .. ',\n"modFlag":' .. encodeEnum(ModFlag, "ModFlag") ..
			',\n"keywordFlag":' .. encodeEnum(KeywordFlag, "KeywordFlag") .. "}\n"
	elseif which == "statmap" then
		body = encodeStatMap(LoadModule("Data/SkillStatMap")(makeSkillMod, makeFlagMod, makeSkillDataMod), "skillStatMap") .. "\n"
	else
		body = dumpOverlay()
	end
	local f = assert(io.open(outPath, "w"))
	f:write(body)
	f:close()
end

-- HeadlessWrapper swallows a top-level error's message, so the dump runs under xpcall and says why it refused.
local ok, err = xpcall(main, debug.traceback)
if not ok then
	io.stderr:write(tostring(err), "\n")
	os.exit(1)
end
