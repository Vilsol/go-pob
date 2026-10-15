
local outPath = arg[1]
assert(outPath, "usage: skill_stat_map.lua <out.json>")

local loadDataModule = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").loadDataModule
dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scanSource()
	local locs = {}
	local declared = {}
	local dir = "Data/Skills/"
	local p = io.popen("ls " .. dir)
	for file in p:lines() do
		if file:match("%.lua$") then
			local id = nil
			local inMap = false
			local depth = 0
			local n = 0
			for line in io.lines(dir .. file) do
				n = n + 1
				local skillId = line:match('^skills%["([^"]+)"%]%s*=')
				if skillId then
					id = skillId
					inMap = false
				elseif id and not inMap and line:match("^\tstatMap%s*=%s*{") then
					locs[id] = "src/" .. dir .. file .. ":" .. n
					declared[id] = {}
					inMap = true
					depth = 1
				elseif inMap then
					if line:match("^\t}") then
						inMap = false
						id = nil
					else
						local key = line:match('^\t\t%["([^"]+)"%]%s*=')
						if key then
							declared[id][key] = true
						end
					end
				end
			end
		end
	end
	p:close()
	return locs, declared
end

local locs, declared = scanSource()

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
	loadDataModule("Data/Skills/" .. skillType, freshSkills, makeSkillMod, makeFlagMod, makeSkillDataMod)
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

local function encodeScalars(t)
	local scalarKeys = {}
	for k in pairs(t) do
		if type(k) == "string" then
			table.insert(scalarKeys, k)
		end
	end
	table.sort(scalarKeys)
	local scalars = {}
	for _, k in ipairs(scalarKeys) do
		scalars[k] = encodeValue(t[k])
	end
	return scalars
end

local skills = {}
local skillCount, entryCount, modCount, groupCount = 0, 0, 0, 0

for skillId, grantedEffect in pairs(freshSkills) do
	local map = grantedEffect.statMap
	if map and next(map) then
		local entries = {}
		local n = 0
		for stat, entry in pairs(map) do
			local mods = {}
			local groups = {}
			for _, modOrGroup in ipairs(entry) do
				if modOrGroup.name then
					table.insert(mods, encodeMod(modOrGroup))
					modCount = modCount + 1
				else
					local groupMods = {}
					for _, m in ipairs(modOrGroup) do
						table.insert(groupMods, encodeMod(m))
						modCount = modCount + 1
					end
					table.insert(groups, { mods = groupMods, scalars = encodeScalars(modOrGroup) })
					groupCount = groupCount + 1
				end
			end
			entries[stat] = { mods = mods, scalars = encodeScalars(entry), groups = groups }
			n = n + 1
			entryCount = entryCount + 1
		end
		if n > 0 then
			skills[skillId] = {
				loc = locs[skillId],
				support = grantedEffect.support and true or false,
				entries = entries,
			}
			skillCount = skillCount + 1
		end
	end
end

local missing, metaLeak = {}, {}
for skillId, keys in pairs(declared) do
	for key in pairs(keys) do
		if not (skills[skillId] and skills[skillId].entries[key]) then
			table.insert(missing, skillId .. "/" .. key)
		end
	end
end
for skillId, skill in pairs(skills) do
	for key in pairs(skill.entries) do
		if not (declared[skillId] and declared[skillId][key]) then
			table.insert(metaLeak, skillId .. "/" .. key)
		end
	end
end
table.sort(missing)
table.sort(metaLeak)

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
	skillTypeNames = skillTypeNames,
	modFlagBits = flagBits(ModFlag),
	keywordFlagBits = flagBits(KeywordFlag),
	skillCount = skillCount,
	entryCount = entryCount,
	modCount = modCount,
	groupCount = groupCount,
	sourceScanSkillCount = (function() local c = 0 for _ in pairs(declared) do c = c + 1 end return c end)(),
	missingFromRuntime = missing,
	metatableLeak = metaLeak,
	skills = skills,
}, { indent = true }))
fh:close()

io.stderr:write(string.format(
	"oracle: %d skills with statMap, %d entries, %d mods, %d groups -> %s\n",
	skillCount, entryCount, modCount, groupCount, outPath))
if #missing > 0 then
	io.stderr:write("oracle: MISSING FROM RUNTIME: " .. table.concat(missing, ", ") .. "\n")
end
if #metaLeak > 0 then
	io.stderr:write("oracle: METATABLE LEAK: " .. table.concat(metaLeak, ", ") .. "\n")
end
