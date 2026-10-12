assert(arg[1], "usage: full_dps_skill_breakdown.lua <build.xml> [build.xml ...]")

local paths = {}
for i = 1, #arg do
	paths[i] = arg[i]
end

dofile("HeadlessWrapper.lua")

for i = 1, #paths do
	local xmlPath = paths[i]
	local fh = assert(io.open(xmlPath, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	wipeGlobalCache()
	loadBuildFromXML(xmlText, xmlPath)
	runCallback("OnFrame")

	local out = build.calcsTab.mainEnv.player.output
	print(string.format("BUILD\t%s", xmlPath:match("([^/]+)%.xml$") or xmlPath))
	print(string.format("  FullDPS\t%.8f", out.FullDPS or 0))
	print(string.format("  TotalDPS\t%.8f  (main skill)", out.TotalDPS or 0))
	for _, s in ipairs(out.SkillDPS or {}) do
		print(string.format("  SKILL\t%-28s dps=%-20.8f count=%-6s part=%s",
			tostring(s.name), s.dps or 0, tostring(s.count), tostring(s.skillPart or s.part)))
	end
	print("")
end
