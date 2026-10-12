local xmlPath = arg[1]
assert(xmlPath, "usage: full_dps_pointer_swap.lua <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local calcs = LoadModule("Modules/Calcs")

local function skillName(sk)
	return sk and sk.activeEffect and sk.activeEffect.grantedEffect
		and sk.activeEffect.grantedEffect.name or "?"
end

print("=== C: ONE pointer swap on a FRESH initEnv, no preceding iterations ===")
for i = 1, #build.calcsTab.mainEnv.player.activeSkillList do
	local env = calcs.initEnv(build, "CALCULATOR")
	local sk = env.player.activeSkillList[i]
	if sk and sk.socketGroup and sk.socketGroup.includeInFullDPS then
		env.player.mainSkill = sk
		calcs.perform(env, true)
		print(string.format("  idx=%-3d %-26s TotalDPS=%.8f ImpaleDPS=%.8f",
			i, skillName(sk), env.player.output.TotalDPS or 0,
			env.player.output.ImpaleDPS or 0))
	end
end
