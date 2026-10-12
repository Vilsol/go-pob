local paths = { unpack(arg) }
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

for i = 1, #paths do
	local fh = assert(io.open(paths[i], "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, paths[i])
	runCallback("OnFrame")
	local env = build.calcsTab.mainEnv
	local player = env.player.mainSkill
	local minion = env.minion and env.minion.mainSkill
	print(json.encode({ file = paths[i]:match("[^/]+$"),
		info = player and player.infoMessage or "", info2 = player and player.infoMessage2 or "",
		minionInfo = minion and minion.infoMessage or "", minionInfo2 = minion and minion.infoMessage2 or "" }))
end
