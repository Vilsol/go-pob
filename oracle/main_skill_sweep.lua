local xmlPath = arg[1]
assert(xmlPath, "usage: main_skill_sweep.lua <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

print("=== FullDPS SkillDPS table (Calcs.lua:185 pointer-swap arm) ===")
print(string.format("FullDPS %.8f", build.calcsTab.mainEnv.player.output.FullDPS or 0))
for _, s in ipairs(build.calcsTab.mainEnv.player.output.SkillDPS or {}) do
	print(string.format("  %-24s dps=%.8f count=%s part=%s",
		tostring(s.name), s.dps or 0, tostring(s.count), tostring(s.part)))
end

print("")
print("=== each (socketGroup, activeSkill) SET AS MAIN (full re-init arm) ===")
for gi, group in ipairs(build.skillsTab.socketGroupList) do
	for si = 1, #(group.displaySkillList or {}) do
		build.mainSocketGroup = gi
		group.mainActiveSkill = si
		build.buildFlag = true
		runCallback("OnFrame")
		local env = build.calcsTab.mainEnv
		local ms = env.player.mainSkill
		local name = ms and ms.activeEffect and ms.activeEffect.grantedEffect
			and ms.activeEffect.grantedEffect.name or "?"
		print(string.format("  g%d s%d  main=%-26s TotalDPS=%.8f ImpaleDPS=%.8f inFullDPS=%s",
			gi, si, name, env.player.output.TotalDPS or 0, env.player.output.ImpaleDPS or 0,
			tostring(group.includeInFullDPS)))
	end
end
