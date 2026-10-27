-- Prints every entry of PoB-PoE2's loaded data.skills as `id<TAB>path<TAB>value` rows, for the Go-side
-- differential in skills_pob_test.go. Skill types print by name. A stat set's statMap prints after every
-- stat of its stats and constantStats was read through it, so the fallback to data.skillStatMap and its
-- processMod run as the calc would make them run; hasGlobalEffect prints before and after those reads.
dofile("HeadlessWrapper.lua")

local skillTypeName = {}
for name, id in pairs(SkillType) do
	skillTypeName[id] = name
end

local function num(v)
	if v == math.huge then
		return "+Inf"
	elseif v == -math.huge then
		return "-Inf"
	end
	return string.format("%.17g", v)
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

local function key(k)
	if type(k) == "number" then
		return "[" .. num(k) .. "]"
	end
	return '["' .. k .. '"]'
end

local typeLists = { requireSkillTypes = true, addSkillTypes = true, excludeSkillTypes = true, skillTypeList = true }
local typeSets = { skillTypes = true, minionSkillTypes = true }

local function flatten(id, path, v, field)
	local ty = type(v)
	if ty == "number" then
		if field == "skillType" then
			io.write(id, "\t", path, "\t", tostring(skillTypeName[v]), "\n")
		else
			io.write(id, "\t", path, "\t", num(v), "\n")
		end
	elseif ty == "string" then
		io.write(id, "\t", path, "\t", '"', (v:gsub("\n", "\\n")), '"', "\n")
	elseif ty == "boolean" then
		io.write(id, "\t", path, "\t", tostring(v), "\n")
	elseif ty == "function" then
		io.write(id, "\t", path, "\tfunction\n")
	elseif ty == "table" then
		if next(v) == nil then
			io.write(id, "\t", path, "\t{}\n")
			return
		end
		for _, k in ipairs(sortedKeys(v)) do
			if k ~= "_grantedEffect" and k ~= "statMap" then
				local val = v[k]
				if typeSets[field] and type(k) == "number" then
					flatten(id, path .. key(skillTypeName[k] or ("Unknown" .. k)), val)
				elseif typeLists[field] then
					flatten(id, path .. key(k), val, "skillType")
				else
					flatten(id, path .. key(k), val, k)
				end
			end
		end
	end
end

local ids = sortedKeys(data.skills)
for _, id in ipairs(ids) do
	local ge = data.skills[id]
	flatten(id, "", ge)
	io.write(id, "\thasGlobalEffect\t", tostring(ge.hasGlobalEffect == true), "\n")
	for i, set in ipairs(ge.statSets) do
		local names = {}
		for _, stat in ipairs(set.stats or {}) do
			names[#names + 1] = stat
		end
		for _, cs in ipairs(set.constantStats or {}) do
			names[#names + 1] = cs[1]
		end
		for _, stat in ipairs(names) do
			local _ = set.statMap[stat]
		end
		for _, stat in ipairs(sortedKeys(set.statMap)) do
			if stat ~= "_grantedEffect" then
				flatten(id, '["statSets"][' .. i .. ']["statMap"]' .. key(stat), set.statMap[stat])
			end
		end
	end
	io.write(id, "\thasGlobalEffectAfterStatMap\t", tostring(ge.hasGlobalEffect == true), "\n")
end
io.write("#count\t\t", #ids, "\n")
