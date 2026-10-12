local path, depth = arg[1], tonumber(arg[2])
assert(path and depth, "usage: power_combined.lua <build.xml> <depth>")
dofile("HeadlessWrapper.lua")

local fh = assert(io.open(path, "r"))
local xmlText = fh:read("*a")
fh:close()
wipeGlobalCache()
loadBuildFromXML(xmlText, path)
runCallback("OnFrame")

local calcsTab = build.calcsTab
calcsTab.powerStat = data.powerStatList[1]
assert(calcsTab.powerStat.combinedOffDef, "the first power stat is not Offence/Defence")
calcsTab.nodePowerMaxDepth = depth
calcsTab:PowerBuilder()

local ids = {}
for id, node in pairs(build.spec.nodes) do
	if not build.spec.allocNodes[id] and node.type ~= "Mastery" and node.power and node.power.offence then
		ids[#ids + 1] = id
	end
end
table.sort(ids)
for _, id in ipairs(ids) do
	local p = build.spec.nodes[id].power
	print(string.format("%d\t%.17g\t%.17g", id, p.offence, p.defence))
end
