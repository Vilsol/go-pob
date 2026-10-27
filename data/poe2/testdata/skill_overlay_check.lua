-- Prints the hand-written half of PoB-PoE2's loaded data.skills, data.skillStatMap and the Global.lua
-- enums as `kind<TAB>key<TAB>field<TAB>value` rows, read off the tables Data.lua builds (not the
-- generator's re-load), for the Go-side differential in skill_overlay_pob_test.go. Load-time additions
-- the generated tables leave to the calc are taken back out: a mod's source, and the notMinionStat tag
-- processMod appends.
dofile("HeadlessWrapper.lua")

local function out(kind, key, field, v)
	io.write(kind, "\t", key, "\t", field, "\t", (v:gsub("\n", "\\n")), "\n")
end

local skillTypeName = {}
for name, id in pairs(SkillType) do
	skillTypeName[id] = name
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

local function isMod(t)
	return type(t.name) == "string" and t.flags ~= nil and t.keywordFlags ~= nil
end

local function canon(v, inTag)
	local ty = type(v)
	if ty == "nil" then
		return "nil"
	elseif ty == "number" then
		if v == math.huge then
			return "+Inf"
		elseif v == -math.huge then
			return "-Inf"
		end
		return string.format("%.17g", v)
	elseif ty == "string" then
		return '"' .. v .. '"'
	elseif ty == "boolean" then
		return tostring(v)
	end
	local mod = isMod(v)
	local parts = {}
	for _, k in ipairs(sortedKeys(v)) do
		if not (mod and k == "source") then
			local val = v[k]
			if inTag and k == "skillType" and type(val) == "number" then
				val = skillTypeName[val]
			elseif inTag and k == "skillTypeList" then
				local names = {}
				for i, id in ipairs(val) do
					names[i] = skillTypeName[id]
				end
				val = names
			end
			local key = type(k) == "number" and "[" .. k .. "]" or k
			parts[#parts + 1] = key .. "=" .. canon(val, inTag or (mod and type(k) == "number"))
		end
	end
	return "{" .. table.concat(parts, ",") .. "}"
end

local function canonMod(m, dropTag)
	local tags = {}
	for i, tag in ipairs(m) do
		tags[i] = canon(tag, true)
	end
	if dropTag then
		local last = m[#m]
		assert(last and last.type == "ActorCondition" and last.actor == "parent" and last.neg, "the notMinionStat tag is not last")
		tags[#tags] = nil
	end
	return m.name .. "|" .. canon(m.type) .. "|" .. canon(m.value) .. "|" .. canon(m.flags) .. "|" .. canon(m.keywordFlags) .. "|" .. table.concat(tags, ";")
end

local function scalars(t)
	return "value=" .. canon(t.value) .. ",mult=" .. canon(t.mult) .. ",div=" .. canon(t.div) .. ",base=" .. canon(t.base)
end

-- Data.lua:85-97, the condition under which processMod appended the tag.
local function notMinionStat(ge, stat)
	if not (ge.support or ge.skillTypes and ge.skillTypes[SkillType.Buff]) then
		return false
	end
	for _, set in ipairs(ge.statSets) do
		for _, name in ipairs(set.notMinionStat or {}) do
			if name == stat then
				return true
			end
		end
	end
	return false
end

local function canonEntry(entry, drop)
	local items = {}
	for i, modOrGroup in ipairs(entry) do
		if modOrGroup.name then
			items[i] = canonMod(modOrGroup, drop)
		else
			local mods = {}
			for j, m in ipairs(modOrGroup) do
				mods[j] = canonMod(m, drop)
			end
			items[i] = "group(" .. scalars(modOrGroup) .. ")[" .. table.concat(mods, " ; ") .. "]"
		end
	end
	return scalars(entry) .. ",skillFlag=" .. canon(entry.skillFlag) .. "|" .. table.concat(items, " ; ")
end

local function sortedSet(t)
	if t == nil then
		return "nil"
	end
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys)
	return table.concat(keys, ",")
end

for _, enum in ipairs({ "SkillType", "ModFlag", "KeywordFlag" }) do
	local n = 0
	for name, v in pairs(_G[enum]) do
		n = n + 1
		out("enum", enum, name, string.format("%.17g", v))
	end
	out("count", "enum:" .. enum, "", tostring(n))
end

local nStat = 0
for stat, entry in pairs(data.skillStatMap) do
	nStat = nStat + 1
	out("statmap", stat, "entry", canonEntry(entry, false))
end
out("count", "statmap", "", tostring(nStat))

local nSkill = 0
for id, ge in pairs(data.skills) do
	nSkill = nSkill + 1
	for _, f in ipairs({ "hidden", "fromItem", "fromTree", "hideFromSideBar", "grantsThornsDamage", "legacy", "minionHasItemSet" }) do
		out("skill", id, f, tostring(ge[f] == true))
	end
	out("skill", id, "preDamageFunc", tostring(ge.preDamageFunc ~= nil))
	out("skill", id, "minionList", ge.minionList and table.concat(ge.minionList, ",") or "")
	out("skill", id, "addFlags", ge.addFlags and sortedSet(ge.addFlags) or "")
	out("skill", id, "minionUses", ge.minionUses and sortedSet(ge.minionUses) or "")
	out("skill", id, "sets", tostring(#ge.statSets))
	for i, set in ipairs(ge.statSets) do
		local key = id .. "#" .. i
		out("set", key, "baseFlags", sortedSet(set.baseFlags))
		local mods = {}
		for j, m in ipairs(set.baseMods or {}) do
			mods[j] = canonMod(m, false)
		end
		out("set", key, "baseMods", set.baseMods and table.concat(mods, " ; ") or "nil")
		local stats = {}
		for stat, entry in pairs(set.statMap) do
			if type(stat) == "string" and stat ~= "_grantedEffect" then
				stats[#stats + 1] = stat
				out("set", key, "stat:" .. stat, canonEntry(entry, notMinionStat(ge, stat)))
			end
		end
		table.sort(stats)
		out("set", key, "stats", table.concat(stats, ","))
	end
end
out("count", "skill", "", tostring(nSkill))
