local pathA, pathB, stat = arg[1], arg[2], arg[3]
assert(pathA and pathB and stat, "usage: compare_power.lua <a.xml> <b.xml> <stat>")
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
if not pcall(require, "zlib") then
	package.preload["zlib"] = function()
		local refuse = function()
			error("compare_power.lua: lua-zlib is not installed; pick builds without a timeless jewel")
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

local function read(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end

wipeGlobalCache()
loadBuildFromXML(read(pathA), pathA)
runCallback("OnFrame")

local compareTab = build.compareTab
assert(compareTab:ImportBuild(read(pathB), "B"), "build B did not import")
local entry = compareTab:GetActiveCompare()

local powerStat
for _, s in ipairs(data.powerStatList) do
	if s.stat == stat then
		powerStat = s
	end
end
assert(powerStat, "no such power stat: " .. stat)

local multiplier = 1
for _, ds in ipairs(build.displayStats) do
	if ds.stat == stat then
		multiplier = (ds.pc or ds.mod) and 100 or 1
		break
	end
end

compareTab:ComparePowerBuilder(entry, powerStat, { treeNodes = true, items = true, skillGems = true, supportGems = true, config = true })

local rows = {}
for _, r in ipairs(compareTab.comparePowerResults or {}) do
	rows[#rows + 1] = string.format("%s\t%s\t%.17g", r.category, r.name, r.impact / multiplier)
end
table.sort(rows)
for _, row in ipairs(rows) do
	print(row)
end
