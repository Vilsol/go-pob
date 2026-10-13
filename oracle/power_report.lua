local args = {}
for i = 1, #arg do
	args[i] = arg[i]
end
local outDir, xmlPath = args[1], args[2]
assert(outDir and xmlPath and args[3], "usage: power_report.lua <out dir> <build.xml> <stat>:<depth> [...]")
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()
wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local function num(v)
	if v == nil then
		return "nil"
	end
	return string.format("%.17g", v)
end

local buildName = xmlPath:match("([^/]+)%.xml$")
local calcsTab = build.calcsTab
local spec = build.spec
for i = 3, #args do
	local stat, depth = args[i]:match("^(%w+):(%d+)$")
	assert(stat, "bad <stat>:<depth> " .. args[i])
	depth = tonumber(depth)
	local powerStat
	for _, s in ipairs(data.powerStatList) do
		if (stat == "OffDef" and s.combinedOffDef) or s.stat == stat then
			powerStat = s
		end
	end
	assert(powerStat, "no such power stat: " .. stat)
	calcsTab.powerStat = powerStat
	calcsTab.nodePowerMaxDepth = depth > 0 and depth or nil
	calcsTab.powerMax = nil
	calcsTab:PowerBuilder()

	local lines = {}
	local granted = calcsTab.mainEnv.grantedPassives
	for id, node in pairs(spec.nodes) do
		local p = node.power
		if p and next(p) then
			local ids = {}
			for _, n in ipairs(not node.alloc and node.path or {}) do
				ids[#ids + 1] = n.id
			end
			table.sort(ids)
			lines[#lines + 1] = table.concat({ "N", id, node.type, tostring(node.alloc and true or false),
				tostring(granted[id] and true or false), num(p.distance), num(p.singleStat), num(p.pathPower), num(p.offence),
				num(p.defence), #(node.depends or {}), table.concat(ids, ",") }, "\t")
			for effect, ep in pairs(p.masteryEffects or {}) do
				lines[#lines + 1] = table.concat({ "E", id, effect, num(ep.singleStat), num(ep.pathPower), num(ep.offence),
					num(ep.defence) }, "\t")
			end
		end
	end
	for _, node in pairs(spec.tree.clusterNodeMap) do
		if node.power and node.power.singleStat then
			lines[#lines + 1] = table.concat({ "C", node.id, node.dn, num(node.power.singleStat) }, "\t")
		end
	end
	table.sort(lines)
	local m = calcsTab.powerMax
	lines[#lines + 1] = table.concat({ "X", num(m.singleStat), num(m.offence), num(m.defence), num(m.offencePerPoint),
		num(m.defencePerPoint) }, "\t")

	local multiplier = 1
	for _, ds in ipairs(build.displayStats) do
		if ds.stat == powerStat.stat then
			multiplier = (ds.pc or ds.mod) and 100 or 1
			break
		end
	end
	for _, r in ipairs(build.treeTab:BuildPowerReportList(powerStat)) do
		lines[#lines + 1] = table.concat({ "R", r.id, r.type, r.name, num(r.power / multiplier),
			num(r.pathPower / multiplier), tostring(r.pathDist), tostring(r.allocated and true or false) }, "\t")
	end

	local out = assert(io.open(string.format("%s/%s_%s_%d.tsv", outDir, buildName, stat, depth), "w"))
	out:write(table.concat(lines, "\n"), "\n")
	out:close()
	io.stderr:write(string.format("%s %s:%d %d lines\n", buildName, stat, depth, #lines))
end
