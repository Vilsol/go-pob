local paths = {}
for i = 1, #arg do
	paths[i] = arg[i]
end
assert(paths[1], "usage: node_distance_to_class_start.lua <build.xml> [<build.xml> ...]")

dofile("HeadlessWrapper.lua")

for i = 1, #paths do
	local xmlPath = paths[i]
	local fh = assert(io.open(xmlPath, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	wipeGlobalCache()
	loadBuildFromXML(xmlText, xmlPath)
	runCallback("OnFrame")

	local spec = build.calcsTab.mainEnv.spec
	local ids = {}
	for id, node in pairs(spec.nodes) do
		if node.alloc and (node.isJewelSocket or node.expansionJewel) then
			ids[#ids + 1] = id
		end
	end
	table.sort(ids)

	for _, id in ipairs(ids) do
		print(string.format("DIST\t%s\t%s\t%s", xmlPath:match("[^/\\]+$"), tostring(id),
			tostring(spec.nodes[id].distanceToClassStart)))
	end
end
