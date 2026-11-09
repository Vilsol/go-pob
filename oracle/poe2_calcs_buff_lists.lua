-- usage (from PoB-PoE2's src/): luajit poe2_calcs_buff_lists.lua <out.jsonl> <build.xml>...
-- Records the Calcs tab's Aura and Buff Skills, Combat Buffs and Curses and Debuffs lists PoB-PoE2's CALCS
-- calculation writes (output.BuffList, CombatList, CurseList and the minion's), with the main skill of the MAIN and
-- CALCS calculations, one JSON line per build.
local outPath, paths = arg[1], { select(2, unpack(arg)) }
assert(outPath and paths[1], "usage: poe2_calcs_buff_lists.lua <out.jsonl> <build.xml>...")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function skillName(env)
	local s = env.player.mainSkill
	return s and s.activeEffect.grantedEffect.name or nil
end

local out = assert(io.open(outPath, "w"))
for _, path in ipairs(paths) do
	local f = assert(io.open(path))
	local xml = f:read("*a")
	f:close()
	local rec = { build = path:match("([^/\\]+)$") }
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(xml)
		runCallback("OnFrame")
		local calcsTab = build.calcsTab
		local output = calcsTab.calcsOutput
		rec.mainSkill = skillName(calcsTab.mainEnv)
		rec.calcsSkill = skillName(calcsTab.calcsEnv)
		rec.BuffList, rec.CombatList, rec.CurseList = output.BuffList, output.CombatList, output.CurseList
		if output.Minion then
			rec.Minion = { BuffList = output.Minion.BuffList, CombatList = output.Minion.CombatList, CurseList = output.Minion.CurseList }
		end
	end)
	if not ok then
		rec.error = tostring(err)
	end
	out:write(json.encode(rec), "\n")
	out:flush()
	collectgarbage("collect")
end
out:close()
