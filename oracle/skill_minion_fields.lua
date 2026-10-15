
local outPath = arg[1]
assert(outPath, "usage: skill_minion_fields.lua <out.json>")

local loadDataModule = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").loadDataModule
dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scanSource()
	local locs = {}
	local dir = "Data/Skills/"
	local p = assert(io.popen("ls " .. dir))
	for file in p:lines() do
		if file:match("%.lua$") then
			local n = 0
			for line in io.lines(dir .. file) do
				n = n + 1
				local skillId = line:match('^skills%["([^"]+)"%]%s*=')
				if skillId then
					locs[skillId] = "src/" .. dir .. file .. ":" .. n
				end
			end
		end
	end
	p:close()
	return locs
end

local locs = scanSource()

local function makeSkillMod(modName, modType, modVal, flags, keywordFlags, ...)
	return { name = modName, type = modType, value = modVal, flags = flags or 0, keywordFlags = keywordFlags or 0, ... }
end
local function makeFlagMod(modName, ...)
	return makeSkillMod(modName, "FLAG", true, 0, 0, ...)
end
local function makeSkillDataMod(dataKey, dataValue, ...)
	return makeSkillMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
end

local modules = { "act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre",
	"sup_str", "sup_dex", "sup_int" }
local freshSkills = {}
for _, skillType in ipairs(modules) do
	loadDataModule("Data/Skills/" .. skillType, freshSkills, makeSkillMod, makeFlagMod, makeSkillDataMod)
end

assert(next(freshSkills), "REFUSING: LoadModule produced no skills; the module list is stale")

local refusals = {}
local function refuse(msg)
	refusals[#refusals + 1] = msg
end

local minionList, addMinionList, minionUses, minionHasItemSet = {}, {}, {}, {}
local nList, nAdd, nUses, nItemSet, nEmpty = 0, 0, 0, 0, 0

local function copyIdList(t, where)
	local out = {}
	for i, v in ipairs(t) do
		if type(v) ~= "string" then
			refuse(where .. ": entry " .. i .. " is a " .. type(v) .. ", expected a minion id")
		end
		out[i] = v
	end
	local n = 0
	for _ in pairs(t) do n = n + 1 end
	if n ~= #t then
		refuse(where .. ": table has " .. n .. " keys but array length " .. #t)
	end
	return out
end

for skillId, ge in pairs(freshSkills) do
	local where = (locs[skillId] or ("<unlocated> " .. skillId))

	if ge.minionList then
		minionList[skillId] = copyIdList(ge.minionList, where .. " minionList")
		nList = nList + 1
		if #ge.minionList == 0 then
			nEmpty = nEmpty + 1
		end
	end

	if ge.addMinionList then
		addMinionList[skillId] = copyIdList(ge.addMinionList, where .. " addMinionList")
		nAdd = nAdd + 1
	end

	if ge.minionUses then
		local slots = {}
		for slot, on in pairs(ge.minionUses) do
			if type(slot) ~= "string" or type(on) ~= "boolean" then
				refuse(where .. " minionUses: expected <string slot> = <boolean>, got " ..
					type(slot) .. " = " .. type(on))
			else
				slots[slot] = on
			end
		end
		minionUses[skillId] = slots
		nUses = nUses + 1
	end

	if ge.minionHasItemSet ~= nil then
		if type(ge.minionHasItemSet) ~= "boolean" then
			refuse(where .. " minionHasItemSet: expected a boolean, got " ..
				type(ge.minionHasItemSet))
		end
		minionHasItemSet[skillId] = ge.minionHasItemSet
		nItemSet = nItemSet + 1
	end
end

if #refusals > 0 then
	io.stderr:write("REFUSING: the pinned PoB carries shapes this dumper does not model:\n")
	for _, r in ipairs(refusals) do
		io.stderr:write("  " .. r .. "\n")
	end
	os.exit(3)
end

local file = assert(io.open(outPath, "w"))
file:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	locs = locs,
	minionList = minionList,
	addMinionList = addMinionList,
	minionUses = minionUses,
	minionHasItemSet = minionHasItemSet,
}, { indent = true }))
file:close()

io.stderr:write(string.format(
	"wrote %s (PoB %s, %d minionList of which %d empty, %d addMinionList, %d minionUses, %d minionHasItemSet)\n",
	outPath, tostring(launch.versionNumber or "unknown"), nList, nEmpty, nAdd, nUses, nItemSet))
