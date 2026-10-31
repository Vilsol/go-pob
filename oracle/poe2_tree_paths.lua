-- usage (from PoB-PoE2's src/): luajit poe2_tree_paths.lua <out.json> <build.xml>...
-- Records, per build, the active spec as PassiveSpec:Save writes it, the nodes the calc granted, every
-- node's path length, and the spec PassiveSpec:Save writes after AllocNode on a spread of unallocated
-- nodes and after DeallocNode on a spread of allocated ones, each from the loaded state.
local selfPath, outPath = arg[0], arg[1]
assert(outPath and arg[2], "usage: poe2_tree_paths.lua <out.json> <build.xml>...")
local paths = {}
for i = 2, #arg do
	paths[#paths + 1] = arg[i]
end

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function save(spec)
	local xml = {}
	spec:Save(xml)
	local out = { nodes = xml.attrib.nodes, classId = xml.attrib.classId, ascendClassId = xml.attrib.ascendClassId, weaponSets = {} }
	for _, child in ipairs(xml) do
		if child.elem and child.elem:match("^WeaponSet") then
			out.weaponSets[child.elem] = child.attrib.nodes
		elseif child.elem == "Overrides" and child[1] then
			out.strNodes, out.dexNodes, out.intNodes = child[1].attrib.strNodes, child[1].attrib.dexNodes, child[1].attrib.intNodes
		end
	end
	return out
end

local function sortedIds(t)
	local ids = {}
	for id in pairs(t) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	return ids
end

local function spread(list, n)
	local out = {}
	if #list == 0 then
		return out
	end
	local step = math.max(1, math.floor(#list / n))
	for i = 1, #list, step do
		out[#out + 1] = list[i]
		if #out == n then
			break
		end
	end
	return out
end

local function snapshot(spec)
	local state = { nodes = {}, allocNodes = {}, hashOverrides = {} }
	for id, node in pairs(spec.nodes) do
		state.nodes[id] = { node.alloc, node.allocMode }
	end
	for id, node in pairs(spec.allocNodes) do
		state.allocNodes[id] = node
	end
	for id, node in pairs(spec.hashOverrides) do
		state.hashOverrides[id] = node
	end
	return state
end

local function restore(spec, state)
	for id, node in pairs(spec.nodes) do
		node.alloc, node.allocMode = state.nodes[id][1], state.nodes[id][2]
	end
	wipeTable(spec.allocNodes)
	for id, node in pairs(state.allocNodes) do
		spec.allocNodes[id] = node
	end
	wipeTable(spec.hashOverrides)
	for id, node in pairs(state.hashOverrides) do
		spec.hashOverrides[id] = node
	end
	spec:BuildAllDependsAndPaths()
end

local results = {}
for _, path in ipairs(paths) do
	local f = assert(io.open(path))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml)
	runCallback("OnFrame")
	local spec = build.spec

	local row = { build = path:match("([^/\\]+)$"), start = save(spec), granted = {}, leap = 0, dist = {}, alloc = {}, dealloc = {} }
	for _, id in ipairs(sortedIds(spec.allocNodes)) do
		local node = spec.allocNodes[id]
		if node.isGrantedPassive and node.isFreeAllocate then
			row.granted[#row.granted + 1] = id
		end
	end
	local candidates, allocated = {}, {}
	for _, id in ipairs(sortedIds(spec.nodes)) do
		local node = spec.nodes[id]
		if #node.intuitiveLeapLikesAffecting > 0 then
			row.leap = row.leap + 1
		end
		if node.path then
			row.dist[tostring(id)] = #node.path
		end
		if node.type ~= "ClassStart" and node.type ~= "AscendClassStart" then
			if node.alloc then
				allocated[#allocated + 1] = id
			elseif node.path then
				candidates[#candidates + 1] = id
			end
		end
	end

	local state = snapshot(spec)
	for _, id in ipairs(spread(candidates, 8)) do
		spec:AllocNode(spec.nodes[id])
		row.alloc[#row.alloc + 1] = { id = id, spec = save(spec) }
		restore(spec, state)
	end
	for _, id in ipairs(spread(allocated, 4)) do
		spec:DeallocNode(spec.nodes[id])
		row.dealloc[#row.dealloc + 1] = { id = id, spec = save(spec) }
		restore(spec, state)
	end
	results[#results + 1] = row
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(results), { indent = true }))
o:write("\n")
o:close()
