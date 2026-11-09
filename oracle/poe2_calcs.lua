-- usage (from PoB-PoE2's src/): luajit poe2_calcs.lua <build.xml> <out.json> [skill_number=<n>] [misc_buffMode=<mode>]
--   [mainActiveSkillCalcs=<n>]
-- Records the Calcs tab PoB-PoE2 draws for a build over its CALCS calculation, after setting the tab's inputs and the
-- shown socket group's active skill the arguments name: the actor's number, true and string outputs
-- (and its MainHand and OffHand tables'),
-- skill flags and truthy skill data, every {p:mod:n} entry with its combined value, each
-- section's enabled flag, and the text of every shown cell and subsection extra. Keys are the 0-based
-- section:subsection:row:column[:entry] ids the frontend uses.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_calcs.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local calcsTab = build.calcsTab
local set = {}
for i = 3, #arg do
	local k, v = arg[i]:match("^([%w_]+)=(.*)$")
	assert(k, "bad argument " .. arg[i])
	set[k] = v
end
if next(set) then
	-- CalcsTab.lua:43-46, 58-63, 165-168
	if set.skill_number then
		calcsTab.input.skill_number = tonumber(set.skill_number)
	end
	if set.misc_buffMode then
		calcsTab.input.misc_buffMode = set.misc_buffMode
	end
	if set.mainActiveSkillCalcs then
		build.skillsTab.socketGroupList[calcsTab.input.skill_number].mainActiveSkillCalcs = tonumber(set.mainActiveSkillCalcs)
	end
	build.buildFlag = true
	runCallback("OnFrame")
end
local actor = calcsTab.calcsEnv.player
local sections = LoadModule("Modules/CalcSections")

-- dkjson writes tostring's 14 digits; the frontend rounds the value itself, so it gets all 17
local function exact(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return v
	end
	return setmetatable({}, { __tojson = function()
		return string.format("%.17g", v)
	end })
end

local function plain(t, ns)
	local out = {}
	for k, v in pairs(t) do
		if type(v) == "number" then
			out[k] = exact(v)
		elseif v == true or type(v) == "string" then
			out[k] = v
		elseif ns and type(v) == "table" and (k == "MainHand" or k == "OffHand") then
			out[k] = plain(v)
		end
	end
	return out
end
local output = plain(actor.output, true)
local activeEffect = actor.mainSkill.activeEffect
local flags = {}
for k, v in pairs((activeEffect.statSetCalcs or activeEffect.statSet).skillFlags or {}) do
	if v then
		flags[k] = true
	end
end
local skillData = {}
for k, v in pairs(actor.mainSkill.skillData) do
	if v then
		skillData[k] = true
	end
end

-- CalcFormat.lua:46-60, one entry at a time
local function entryValue(sectionData)
	local modCfg = (sectionData.cfg and actor.mainSkill and actor.mainSkill[sectionData.cfg .. "Cfg"]) or {}
	if sectionData.modSource then
		modCfg.source = sectionData.modSource
	end
	if sectionData.actor then
		modCfg.actor = sectionData.actor
	end
	local modStore = (sectionData.enemy and actor.enemy and actor.enemy.modDB) or (sectionData.cfg and actor.mainSkill and actor.mainSkill.skillModList) or actor.modDB
	if type(sectionData.modName) == "table" then
		return modStore:Combine(sectionData.modType, modCfg, unpack(sectionData.modName))
	end
	return modStore:Combine(sectionData.modType, modCfg, sectionData.modName)
end

local mods, entries, cells, extras, enabled = {}, {}, {}, {}, {}
for _, s in ipairs(sections) do
	local id = s[2]
	local sub1 = s[5][1]
	local on = calcsTab:CheckFlag({ flag = sub1.data.flag, notFlag = sub1.data.notFlag }, actor, actor) and true or false
	local any = false
	for j, sub in ipairs(s[5]) do
		for k, row in ipairs(sub.data) do
			for l, col in ipairs(row) do
				local cell = id .. ":" .. (j - 1) .. ":" .. (k - 1) .. ":" .. (l - 1)
				for m, e in ipairs(col) do
					if e.modName and e.modType then
						local v = entryValue(e)
						if type(v) == "number" then
							mods[cell .. ":" .. (m - 1)] = exact(v)
						end
						entries[cell .. ":" .. (m - 1)] = { modName = type(e.modName) == "table" and e.modName or { e.modName }, modType = e.modType, cfg = e.cfg, enemy = e.enemy, modSource = e.modSource, actor = e.actor }
					end
				end
				if on and calcsTab:CheckFlag(row, actor, actor) and col.format and calcsTab:CheckFlag(col, actor, actor) then
					cells[cell] = formatCalcStr(col.format, actor, col)
				end
			end
			if on and calcsTab:CheckFlag(row, actor, actor) then
				any = true
			end
		end
		if on and sub.data.extra then
			extras[id .. ":" .. (j - 1)] = formatCalcStr(sub.data.extra, actor)
		end
	end
	enabled[id] = on and any
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ mainSkill = actor.mainSkill.activeEffect.grantedEffect.name, output = output, skillFlags = flags, skillData = skillData, mods = mods, entries = entries, cells = cells, extras = extras, enabled = enabled }), { indent = true }))
o:write("\n")
o:close()
