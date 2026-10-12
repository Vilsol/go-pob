
local outPath = arg[1]
assert(outPath, "usage: skill_table.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scanSource()
	local locs = {}
	local dir = "Data/Skills/"
	local p = io.popen("ls " .. dir)
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

local freshSkills = {}
for _, skillType in ipairs({ "act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int" }) do
	LoadModule("Data/Skills/" .. skillType, freshSkills, makeSkillMod, makeFlagMod, makeSkillDataMod)
end

local skillTypeName = {}
for name, n in pairs(SkillType) do
	skillTypeName[n] = name
end

local out = {}
local nSkills, nHidden, nSupport, nNoStats, nNoTypes, nWeaponTypes = 0, 0, 0, 0, 0, 0

for skillId, ge in pairs(freshSkills) do
	local types = {}
	if ge.skillTypes then
		for n, on in pairs(ge.skillTypes) do
			if on and skillTypeName[n] then
				table.insert(types, skillTypeName[n])
			end
		end
	end
	table.sort(types)

	local stats = {}
	if ge.stats then
		for i, s in ipairs(ge.stats) do
			stats[i] = s
		end
	end

	local levelCount, maxValues = 0, 0
	if ge.levels then
		for _, lv in pairs(ge.levels) do
			levelCount = levelCount + 1
			if #lv > maxValues then
				maxValues = #lv
			end
		end
	end

	local flags = {}
	if ge.baseFlags then
		for k, on in pairs(ge.baseFlags) do
			if on then
				table.insert(flags, k)
			end
		end
	end
	table.sort(flags)

	local weaponTypes = nil
	if ge.weaponTypes then
		weaponTypes = {}
		for k, on in pairs(ge.weaponTypes) do
			if on then
				table.insert(weaponTypes, k)
			end
		end
		table.sort(weaponTypes)
	end

	out[skillId] = {
		name = ge.name,
		loc = locs[skillId],
		hidden = ge.hidden and true or false,
		support = ge.support and true or false,
		skillTypes = types,
		stats = stats,
		levelCount = levelCount,
		maxLevelValues = maxValues,
		baseFlags = flags,
		weaponTypes = weaponTypes,
	}

	nSkills = nSkills + 1
	if ge.hidden then nHidden = nHidden + 1 end
	if ge.support then nSupport = nSupport + 1 end
	if #stats == 0 then nNoStats = nNoStats + 1 end
	if #types == 0 then nNoTypes = nNoTypes + 1 end
	if weaponTypes then nWeaponTypes = nWeaponTypes + 1 end
end

local fh = assert(io.open(outPath, "w"))
fh:write(json.encode({
	skillCount = nSkills,
	hiddenCount = nHidden,
	supportCount = nSupport,
	noStatsCount = nNoStats,
	noSkillTypesCount = nNoTypes,
	weaponTypesCount = nWeaponTypes,
	skills = out,
}, { indent = false }))
fh:close()

io.stderr:write(string.format(
	"oracle: %d skills (%d hidden, %d support, %d with no stats, %d with no skillTypes, %d with weaponTypes) -> %s\n",
	nSkills, nHidden, nSupport, nNoStats, nNoTypes, nWeaponTypes, outPath))
