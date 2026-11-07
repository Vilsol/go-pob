-- usage (from PoB-PoE2's src/): luajit poe2_sidebar_breakdowns.lua <build.xml> <out.json>
-- Records, for every player display stat (by its index in BuildDisplayStats.lua's displayStats), the breakdown key
-- GetStatBreakdownKey gives it and the cells GetSidebarBreakdown shows for that key whose flags CheckFlag passes:
-- each cell's breakdown, modName list, modType and label, beside the stat's own childStat, breakdown and modNames and
-- the key's breakdownSource. A stat with no key is absent.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_sidebar_breakdowns.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv
local actor = env.player
local out = {}
for index, statData in ipairs(build.displayStats) do
	if statData.stat then
		local key = build:GetStatBreakdownKey(statData, "player")
		if key then
			local ignored
			if statData.ignoredSections then
				ignored = {}
				for _, label in ipairs(statData.ignoredSections) do
					ignored[label] = true
				end
			end
			local cells = {}
			for _, cell in ipairs(build:GetSidebarBreakdown(key, statData.modNames, ignored, "player")) do
				if build.calcsTab:CheckFlag(cell, actor, env.player) then
					local modName = cell.modName
					if type(modName) == "string" then
						modName = { modName }
					end
					cells[#cells + 1] = { breakdown = cell.breakdown, modName = modName, modType = cell.modType, label = cell.label }
				end
			end
			local breakdownData = actor.breakdown and actor.breakdown[key]
			out[#out + 1] = { index = index, stat = statData.stat, childStat = statData.childStat, breakdown = statData.breakdown,
				modNames = statData.modNames, key = key, source = breakdownData and breakdownData.breakdownSource, cells = cells }
		end
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
