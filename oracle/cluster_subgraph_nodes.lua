local paths = { unpack(arg) }
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

for i = 1, #paths do
	local fh = assert(io.open(paths[i], "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, paths[i])
	runCallback("OnFrame")
	local spec = build.spec
	local nodes = { }
	for _, subGraph in pairs(spec.subGraphs) do
		local proxy = tonumber(subGraph.parentSocket.expansionJewel.proxy)
		for _, node in ipairs(subGraph.nodes) do
			local linked = { }
			for _, other in ipairs(node.linked or { }) do
				linked[#linked + 1] = other.id
			end
			table.sort(linked)
			nodes[#nodes + 1] = {
				id = node.id, type = node.type, name = node.dn, icon = node.icon, o = node.o, oidx = node.oidx,
				x = node.x, y = node.y, linked = linked, proxy = proxy,
			}
		end
	end
	table.sort(nodes, function(a, b) return a.id < b.id end)
	print(json.encode({ file = paths[i]:match("[^/]+$"), nodes = nodes }))
end
