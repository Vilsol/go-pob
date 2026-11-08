-- Loads each build and runs CalcsTab:PowerBuilder for each <stat>:<max depth> (depth 0 is All; stat "OffDef" is the
-- combined Offence/Defence entry), one JSON row per run: every calculated node's power and path, the power maxima and
-- the power report rows TreeTab:BuildPowerReportList makes from them (those with a power, and the count of all).
-- usage (from PoB-PoE2's src/): power_oracle.lua <out.jsonl> <stat>:<depth>,... <build.xml>...
local outPath, runsArg = arg[1], arg[2]
assert(outPath and runsArg and arg[3], "usage: power_oracle.lua <out.jsonl> <stat>:<depth>,... <build.xml>...")
local xmlPaths = {}
for i = 3, #arg do
	xmlPaths[#xmlPaths + 1] = arg[i]
end

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function num(v)
	if v == nil then
		return "null"
	end
	if v ~= v or v == math.huge or v == -math.huge then
		return '"' .. tostring(v) .. '"'
	end
	return string.format("%.17g", v)
end
local function readFile(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end

local out = assert(io.open(outPath, "w"))
for _, xmlPath in ipairs(xmlPaths) do
	local name = xmlPath:match("([^/]+)%.xml$")
	loadBuildFromXML(readFile(xmlPath), name)
	local build = __mainObject__.main.modes.BUILD
	build.buildFlag = true
	runCallback("OnFrame")
	local calcsTab, spec = build.calcsTab, build.spec
	for run in runsArg:gmatch("[^,]+") do
		local stat, depth = run:match("^(%w+):(%d+)$")
		assert(stat, "bad <stat>:<depth> " .. run)
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

		local ids = {}
		for id, node in pairs(spec.nodes) do
			if node.power and (node.power.singleStat or node.power.offence) then
				ids[#ids + 1] = id
			end
		end
		table.sort(ids)
		local nodes = {}
		for i, id in ipairs(ids) do
			local node = spec.nodes[id]
			local p = node.power
			-- the path the path power added (or the dependants it removed), which a tie between equal routes can change
			local pathIds = {}
			for _, n in ipairs((node.alloc and node.depends) or node.path or {}) do
				pathIds[#pathIds + 1] = n.id
			end
			table.sort(pathIds)
			nodes[i] = "[" .. table.concat({ id, num(p.distance), num(p.singleStat), num(p.pathPower), num(p.offence), num(p.defence),
				encodeString(table.concat(pathIds, ",")) }, ",") .. "]"
		end
		local max = calcsTab.powerMax
		-- the rows with a power; the rest are counted
		local report, rows = {}, 0
		for _, r in ipairs(build.treeTab:BuildPowerReportList(powerStat)) do
			rows = rows + 1
			if r.power ~= 0 or r.pathPower ~= 0 then
				report[#report + 1] = "[" .. table.concat({ r.id, encodeString(r.name), num(r.power), encodeString(r.powerStr), num(r.pathPower),
					encodeString(r.pathPowerStr), tostring(r.allocated and true or false), encodeString(tostring(r.pathDist)) }, ",") .. "]"
			end
		end
		local cluster = 0
		for _ in pairs(spec.tree.clusterNodeMap) do
			cluster = cluster + 1
		end
		out:write('{"build":', encodeString(name), ',"stat":', encodeString(stat), ',"depth":', depth, ',"clusterNodes":', cluster,
			',"max":[', table.concat({ num(max.singleStat), num(max.offence), num(max.offencePerPoint), num(max.defence),
			num(max.defencePerPoint) }, ","), '],"nodes":[', table.concat(nodes, ","), '],"reportRows":', rows, ',"report":[', table.concat(report, ","), "]}\n")
	end
end
out:close()
