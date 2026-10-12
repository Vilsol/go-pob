
local outPath = arg[1]
assert(outPath, "usage: skill_flags.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local skillFiles = {
	"act_str", "act_dex", "act_int", "other", "glove",
	"minion", "spectre", "sup_str", "sup_dex", "sup_int",
}

local refusals = {}
local function refuse(msg)
	refusals[#refusals + 1] = msg
end

do
	local declared = {}
	for _, f in ipairs(skillFiles) do
		declared[f .. ".lua"] = true
	end
	local p = assert(io.popen("ls Data/Skills/"))
	for file in p:lines() do
		if file:match("%.lua$") and not declared[file] then
			refuse("Data/Skills/" .. file .. " exists but is not in this dumper's skillFiles list " ..
				"(mirror of src/Modules/Data.lua:15-26)")
		end
	end
	p:close()
end

local function makeSkillMod(modName, modType, modVal, flags, keywordFlags, ...)
	return { name = modName, type = modType, value = modVal, flags = flags or 0, keywordFlags = keywordFlags or 0, ... }
end
local function makeFlagMod(modName, ...)
	return makeSkillMod(modName, "FLAG", true, 0, 0, ...)
end
local function makeSkillDataMod(dataKey, dataValue, ...)
	return makeSkillMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
end

local modelledColumns = { "name", "hidden", "support", "fromItem", "fromTree", "baseFlags", "addFlags", "color",
	"legacy", "hideFromGemList", "plusVersionOf" }
local locs, sourceCount = {}, {}
for _, col in ipairs(modelledColumns) do
	sourceCount[col] = 0
end
local sourceIDs = 0
local redeclared = {}
local shadowed = {}
local inShadowed = false

for _, mod in ipairs(skillFiles) do
	local path = "Data/Skills/" .. mod .. ".lua"
	local n = 0
	for line in io.lines(path) do
		n = n + 1
		local skillId = line:match('^skills%["([^"]+)"%]%s*=')
			or line:match('^}%s*skills%["([^"]+)"%]%s*=')
		if skillId then
			if locs[skillId] then
				redeclared[#redeclared + 1] = skillId .. " (" .. locs[skillId] ..
					" -> src/" .. path .. ":" .. n .. ")"
			else
				sourceIDs = sourceIDs + 1
			end
			locs[skillId] = "src/" .. path .. ":" .. n
			shadowed[skillId] = (shadowed[skillId] or 0) + 1
		end
		for _, col in ipairs(modelledColumns) do
			if line:match("^\t" .. col .. " = ") or line:match("^    " .. col .. " = ") then
				sourceCount[col] = sourceCount[col] + 1
			end
		end
	end
end

local freshSkills = {}
for _, mod in ipairs(skillFiles) do
	LoadModule("Data/Skills/" .. mod, freshSkills, makeSkillMod, makeFlagMod, makeSkillDataMod)
end

assert(next(freshSkills), "REFUSING: LoadModule produced no skills from src/Data/Skills/*.lua")

local function flagList(t, where, what)
	if t == nil then
		return nil
	end
	if type(t) ~= "table" then
		refuse(where .. ": " .. what .. " is a " .. type(t) .. ", not a table")
		return nil
	end
	local out = {}
	for k, on in pairs(t) do
		if type(k) ~= "string" then
			refuse(where .. ": " .. what .. " key is a " .. type(k) .. ", not a string")
		elseif on ~= true then
			refuse(where .. ": " .. what .. "." .. tostring(k) .. " is " .. tostring(on) ..
				", not the literal true the Go []string form assumes")
		else
			out[#out + 1] = k
		end
	end
	table.sort(out)
	return out
end

local function boolean(v, where, what)
	if v == nil then
		return false
	end
	if v ~= true then
		refuse(where .. ": " .. what .. " is " .. tostring(v) ..
			", not the literal true this dumper and the Go bool field assume")
		return false
	end
	return true
end

local function colorOf(v, where)
	if v == nil then
		return nil
	end
	if type(v) ~= "number" or v % 1 ~= 0 or v < 1 or v > 4 then
		refuse(where .. ": color is " .. tostring(v) ..
			", not an integer 1-4; SkillsTab.lua:1236 indexes { \"R\", \"G\", \"B\" } with it")
		return nil
	end
	return v
end

local function skillIdOf(v, where)
	if v == nil then
		return nil
	end
	if type(v) ~= "string" or v == "" then
		refuse(where .. ": plusVersionOf is " .. tostring(v) .. ", not a skill id string")
		return nil
	end
	return v
end

local out = {}
local loaded = {}
for _, col in ipairs(modelledColumns) do
	loaded[col] = 0
end
local nSkills, nUnnamed = 0, 0
local flagNames, addFlagNames = {}, {}

for skillId, ge in pairs(freshSkills) do
	local where = "src/Data/Skills " .. skillId

	if type(ge.name) ~= "string" then
		refuse(where .. ": name is a " .. type(ge.name) ..
			", not a string; ModParser.lua:1610 keys gemIdLookup on it")
	elseif ge.name == "" then
		nUnnamed = nUnnamed + 1
	end
	if not locs[skillId] then
		refuse(where .. ": no `skills[\"" .. skillId .. "\"] =` line to cite")
	end

	local base = flagList(ge.baseFlags, where, "baseFlags")
	local add = flagList(ge.addFlags, where, "addFlags")
	for _, f in ipairs(base or {}) do
		flagNames[f] = true
	end
	for _, f in ipairs(add or {}) do
		addFlagNames[f] = true
	end

	out[skillId] = {
		id = skillId,
		loc = locs[skillId],
		name = ge.name,
		color = colorOf(ge.color, where),
		hidden = boolean(ge.hidden, where, "hidden"),
		support = boolean(ge.support, where, "support"),
		fromItem = boolean(ge.fromItem, where, "fromItem"),
		fromTree = boolean(ge.fromTree, where, "fromTree"),
		legacy = boolean(ge.legacy, where, "legacy"),
		hideFromGemList = boolean(ge.hideFromGemList, where, "hideFromGemList"),
		plusVersionOf = skillIdOf(ge.plusVersionOf, where),
		baseFlags = base,
		addFlags = add,
	}

	nSkills = nSkills + 1
	for _, col in ipairs(modelledColumns) do
		if ge[col] ~= nil then
			loaded[col] = loaded[col] + 1
		end
	end
end

if nSkills ~= sourceIDs then
	refuse(string.format("loaded %d skills but the source declares %d `skills[\"...\"] =` lines",
		nSkills, sourceIDs))
end
local shadowedCount = #redeclared
for _, col in ipairs(modelledColumns) do
	local delta = sourceCount[col] - loaded[col]
	if delta < 0 or delta > shadowedCount then
		refuse(string.format(
			"column %s: %d rows carry it in the loaded table but %d `\\t%s = ` lines exist in the " ..
			"source (delta %d, %d redeclared skill(s) allow at most that many). Equal-and-wrong " ..
			"is impossible here; unequal means one of the two scans is.",
			col, loaded[col], sourceCount[col], col, delta, shadowedCount))
	end
end
if shadowedCount > 0 then
	io.stderr:write(string.format(
		"note: %d skill id(s) redeclared upstream; Lua's last writer wins and the citation " ..
		"follows it:\n", shadowedCount))
	for _, r in ipairs(redeclared) do
		io.stderr:write("  " .. r .. "\n")
	end
end

if #refusals > 0 then
	io.stderr:write("REFUSING: the pinned PoB carries shapes this dumper does not model:\n")
	for _, r in ipairs(refusals) do
		io.stderr:write("  " .. r .. "\n")
	end
	io.stderr:write("Emitting anyway would silently drop them from the generated Go, and nothing " ..
		"downstream would notice. Model them here and in oracle/skill_flags_emit.py.\n")
	os.exit(3)
end

local function sortedKeys(t)
	local out = {}
	for k in pairs(t) do
		out[#out + 1] = k
	end
	table.sort(out)
	return out
end

local file = assert(io.open(outPath, "w"))
file:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	skillCount = nSkills,
	unnamedCount = nUnnamed,
	columnCounts = loaded,
	baseFlagNames = sortedKeys(flagNames),
	addFlagNames = sortedKeys(addFlagNames),
	skills = out,
}, { indent = true }))
file:close()

io.stderr:write(string.format(
	"wrote %s (PoB %s, %d skills, %d unnamed; %d baseFlags, %d addFlags, %d hidden, %d support, %d fromItem, %d fromTree, %d color)\n",
	outPath, tostring(launch.versionNumber or "unknown"), nSkills, nUnnamed,
	loaded.baseFlags, loaded.addFlags, loaded.hidden, loaded.support, loaded.fromItem, loaded.fromTree,
	loaded.color))
