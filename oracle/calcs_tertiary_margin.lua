local buildsDir = arg[1]
local files = { select(2, unpack(arg)) }
assert(buildsDir and #files > 0, "usage: calcs_tertiary_margin.lua <buildsDir> <build.xml>...")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

for _, file in ipairs(files) do
	local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	loadBuildFromXML(xmlText, file)
	runCallback("OnFrame")
	local env = build.calcsTab.calcsEnv
	local lines = { }
	for _, l in ipairs(env.player.breakdown.AreaOfEffectRadiusTertiary or { }) do
		lines[#lines + 1] = l
	end
	print(json.encode({ build = file, skill = env.player.mainSkill.activeEffect.grantedEffect.name, lines = lines }, { keyorder = { "build", "skill", "lines" } }))
end
