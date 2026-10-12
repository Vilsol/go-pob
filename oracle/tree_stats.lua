
local outPath = arg[1]
assert(outPath, "usage: tree_stats.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local treeVersion = (arg[2] and arg[2] ~= "") and arg[2] or latestTreeVersion
local tree = main:LoadTree(treeVersion)

local out = {}
for id, node in pairs(tree.nodes) do
	if node.sd and #node.sd > 0 then
		local stats = {}
		for i, line in ipairs(node.sd) do
			stats[i] = line
		end
		out[tostring(id)] = {
			name = node.dn,
			stats = stats,
			ascendancy = node.ascendancyName,
			group = node.g,
		}
	end
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode({ version = treeVersion, nodes = out }, { indent = false }))
f:close()
print("wrote " .. outPath)
