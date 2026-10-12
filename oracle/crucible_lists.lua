local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local t_insert = table.insert
local nodes = main.tree and main.tree.nodes or (data.latestTreeVersion and nil)

local function checkLineForAllocates(line, treeNodes)
	if treeNodes and string.match(line, "Allocates") then
		local nodeId = tonumber(string.match(line, "%d+"))
		if treeNodes[nodeId] then return "Allocates "..treeNodes[nodeId].name end
	end
	return line
end

local function build(item, treeNodes)
	local modList = { {}, {}, {}, {}, {} }
	local itemModMap, nodeSelections = {}, {}
	for _, mod in ipairs(item.crucibleModLines) do itemModMap[mod.line] = true end
	for i, mod in pairs(data.crucible) do
		if item:CanHaveMod(mod) then
			if itemModMap[checkLineForAllocates(mod[1], treeNodes)] and ((mod[2] and itemModMap[checkLineForAllocates(mod[2], treeNodes)]) or not mod[2]) then
				if nodeSelections[mod.nodeLocation[1]] and mod.nodeLocation[2] then nodeSelections[mod.nodeLocation[2]] = i
				else nodeSelections[mod.nodeLocation[1]] = i end
			end
			local label = {}
			for index, line in ipairs(mod) do label[index] = checkLineForAllocates(line, treeNodes) end
			for _, location in ipairs(mod.nodeLocation) do
				t_insert(modList[location], { label = table.concat(label, "/") .. " - Tier: " .. mod.tier, affixType = mod.type, defaultOrder = i })
			end
		end
	end
	local out = {}
	for k, tierList in ipairs(modList) do
		table.sort(tierList, function(a, b)
			if a.affixType ~= b.affixType then return a.affixType == "Spawn" and b.affixType == "MergeOnly" else return a.defaultOrder < b.defaultOrder end
		end)
		local labels = {}
		for j, e in ipairs(tierList) do labels[j] = e.label end
		out[k] = { labels = labels, selected = nodeSelections[k] or "" }
	end
	return out
end

local treeNodes = new("PassiveTree", latestTreeVersion).nodes
local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, raw in ipairs(items) do
	out[k] = build(new("Item", raw), treeNodes)
end
io.open(outPath, "w"):write(json.encode(out))
