
local buildsDir, outPath, listFile = arg[1], arg[2], arg[3]
assert(buildsDir and outPath, "usage: dump_speed_rungs.lua <buildsDir> <outFile> [listFile]")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local builds = {}
if listFile then
	for line in assert(io.lines(listFile)) do
		local name = line:match("^%s*(.-)%s*$")
		if name ~= "" and not name:match("^#") then
			builds[#builds + 1] = name
		end
	end
else
	for file in lfs.dir(buildsDir) do
		if file:match("%.xml$") then
			builds[#builds + 1] = file
		end
	end
end
table.sort(builds)

local out = assert(io.open(outPath, "w"))

for _, file in ipairs(builds) do
	local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		local env = build.calcsTab.mainEnv
		local actor = env.player
		local activeSkill = actor.mainSkill
		local skillModList = activeSkill.skillModList
		local cfg = activeSkill.skillCfg

		entry.skill = activeSkill.activeEffect.grantedEffect.name
		entry.isAttack = activeSkill.skillFlags.attack and true or false
		entry.attackSpeedMultiplier = activeSkill.skillData.attackSpeedMultiplier
		entry.speed = actor.output.Speed
		entry.time = actor.output.Time
		entry.repeats = actor.output.Repeats
		entry.actionSpeedMod = actor.output.ActionSpeedMod

		if actor.output.MainHand then
			entry.mhSpeed = actor.output.MainHand.Speed
			entry.mhTime = actor.output.MainHand.Time
		end

		entry.inc = skillModList:Sum("INC", cfg, "Speed")
		entry.more = skillModList:More(cfg, "Speed")
		entry.baseSpeedSum = skillModList:Sum("BASE", cfg, "Speed")
		entry.skillAttackTime = calcLib.mod(skillModList, cfg, "SkillAttackTime")
		entry.totalAttackTime = skillModList:Sum("BASE", cfg, "TotalAttackTime")
		entry.totalCastTime = skillModList:Sum("BASE", cfg, "TotalCastTime")
		entry.castTime = activeSkill.activeEffect.grantedEffect.castTime

		if actor.weaponData1 then
			entry.attackRate1 = actor.weaponData1.AttackRate
			entry.attackSpeedInc1 = actor.weaponData1.AttackSpeedInc
		end
		if actor.weaponData2 then
			entry.attackRate2 = actor.weaponData2.AttackRate
		end
	end)
	if not ok then
		entry.error = tostring(err)
	end
	out:write(json.encode(entry) .. "\n")
	io.stderr:write("speed-rungs: " .. file .. (entry.error and ("  ERROR " .. entry.error) or "") .. "\n")
end
out:close()
io.stderr:write("speed-rungs: wrote " .. #builds .. " builds to " .. outPath .. "\n")
