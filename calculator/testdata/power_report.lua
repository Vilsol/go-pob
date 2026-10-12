local path, stat = arg[1], arg[2]
assert(path and stat, "usage: power_report.lua <build.xml> <stat>")
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
if not pcall(require, "zlib") then
	package.preload["zlib"] = function()
		local refuse = function()
			error("power_report.lua: lua-zlib is not installed; pick a build without a timeless jewel")
		end
		return { inflate = refuse, deflate = refuse }
	end
end
if not pcall(require, "lua-utf8") then
	package.preload["lua-utf8"] = function()
		return setmetatable({ next = function() return nil end }, { __index = string })
	end
end
dofile("HeadlessWrapper.lua")

local fh = assert(io.open(path, "r"))
local xmlText = fh:read("*a")
fh:close()
wipeGlobalCache()
loadBuildFromXML(xmlText, path)
runCallback("OnFrame")

local powerStat
for _, s in ipairs(data.powerStatList) do
	if s.stat == stat then
		powerStat = s
	end
end
assert(powerStat, "no such power stat: " .. stat)

local calcsTab = build.calcsTab
calcsTab.powerStat = powerStat
calcsTab.nodePowerMaxDepth = nil
calcsTab:PowerBuilder()

local multiplier = 1
for _, ds in ipairs(build.displayStats) do
	if ds.stat == stat then
		multiplier = (ds.pc or ds.mod) and 100 or 1
		break
	end
end

local report = build.treeTab:BuildPowerReportList(powerStat)
local rows = {}
for _, r in ipairs(report) do
	if type(r.pathDist) == "number" then
		local node = build.spec.nodes[r.id]
		local ids = {}
		for _, n in ipairs(node and not node.alloc and node.path or {}) do
			ids[#ids + 1] = n.id
		end
		table.sort(ids)
		rows[#rows + 1] = string.format("%d\t%s\t%s\t%.17g\t%.17g\t%d\t%s\t%s", r.id, r.type, r.name, r.power / multiplier,
			r.pathPower / multiplier, r.pathDist, tostring(r.allocated and true or false), table.concat(ids, ","))
	end
end
table.sort(rows)
for _, row in ipairs(rows) do
	print(row)
end
