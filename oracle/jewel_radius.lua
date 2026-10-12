
local selfPath = arg[0]
local outPath = arg[1]
assert(outPath, "usage: jewel_radius.lua <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local tree = main:LoadTree(latestTreeVersion)

local function sortedKeys(t)
	local ids = {}
	for id in pairs(t or {}) do
		table.insert(ids, id)
	end
	table.sort(ids)
	return ids
end

local sockets = {}
for socketId, socket in pairs(tree.sockets) do
	local perRadius = {}
	for radiusIndex = 1, #data.jewelRadius do
		perRadius[radiusIndex] = sortedKeys(socket.nodesInRadius and socket.nodesInRadius[radiusIndex])
	end
	sockets[tostring(socketId)] = {
		x = socket.x,
		y = socket.y,
		nodesInRadius = perRadius,
	}
end

local jewelRadius = {}
for i, r in ipairs(data.jewelRadius) do
	jewelRadius[i] = { inner = r.inner, outer = r.outer, label = r.label }
end

local nodePositions = {}
for nodeId, node in pairs(tree.nodes) do
	if node.x and node.y then
		nodePositions[tostring(nodeId)] = {
			x = node.x,
			y = node.y,
			ascend = node.ascendancyName ~= nil,
			group = node.group and true or false,
		}
	end
end

local out = {
	treeVersion = latestTreeVersion,
	orbitRadii = tree.orbitRadii,
	skillsPerOrbit = tree.skillsPerOrbit,
	jewelRadius = jewelRadius,
	nodePositions = nodePositions,
	socketCount = 0,
	sockets = sockets,
}
for _ in pairs(sockets) do
	out.socketCount = out.socketCount + 1
end

local fh = assert(io.open(outPath, "w"))
fh:write(json.encode(stable(out), { indent = true }))
fh:close()
io.stderr:write(string.format("oracle: %d sockets, %d radii, tree %s -> %s\n",
	out.socketCount, #data.jewelRadius, latestTreeVersion, outPath))
