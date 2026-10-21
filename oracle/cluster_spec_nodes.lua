-- usage: luajit cluster_spec_nodes.lua <build.xml>:<spec index> ...
-- Prints the cluster nodes in spec.nodes (what PoB allocates and draws), not spec.subGraphs:
-- subGraphs is keyed by node id, so a nested subgraph whose id collides with its parent's replaces it there.
local args = { unpack(arg) }
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

for i = 1, #args do
	local path, specIndex = args[i]:match("^(.+):(%d+)$")
	assert(path, "REFUSING: argument is not <build.xml>:<spec index>: " .. args[i])
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, path)
	runCallback("OnFrame")
	specIndex = tonumber(specIndex)
	assert(build.treeTab.specList[specIndex], "REFUSING: no spec " .. specIndex .. " in " .. path)
	build.treeTab:SetActiveSpec(specIndex)
	runCallback("OnFrame")
	local spec = build.spec
	local proxyAt = { }
	for _, treeNode in pairs(spec.tree.nodes) do
		if treeNode.expansionJewel then
			local proxy = spec.tree.nodes[tonumber(treeNode.expansionJewel.proxy)]
			proxyAt[proxy.group.x .. "," .. proxy.group.y] = proxy.id
		end
	end
	local nodes = { }
	for id, node in pairs(spec.nodes) do
		if id >= 0x10000 or (node.expansionJewel and node.expansionJewel.parent) then
			local linked = { }
			for _, other in ipairs(node.linked or { }) do
				linked[#linked + 1] = other.id
			end
			table.sort(linked)
			local proxy = proxyAt[node.group.x .. "," .. node.group.y]
			assert(proxy, "REFUSING: node " .. id .. " sits in no proxy group")
			nodes[#nodes + 1] = {
				id = node.id, type = node.type, name = node.dn, icon = node.icon, o = node.o, oidx = node.oidx,
				x = node.x, y = node.y, linked = linked, proxy = proxy, sd = node.sd,
			}
		end
	end
	table.sort(nodes, function(a, b) return a.id < b.id end)
	print(json.encode({ file = path:match("[^/]+$"), spec = specIndex, nodes = nodes }))
end
