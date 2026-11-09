-- usage (from PoB-PoE2's src/): luajit poe2_party.lua <out.json> <giver.xml|buffers.txt> <receiver.xml>...
-- Records what PoB-PoE2's party tab exports for the giver build (Import tab's "Export support" on, PartyTab:Save's
-- ExportedBuffs), then imports that export into each receiver's party tab (PartyTab:Load of ImportedBuffs) and
-- records the receiver's player and enemy numeric outputs with and without it. A .txt giver holds the buffers
-- themselves, each under a "=== <name>" line.
local selfPath, outPath, giverPath = arg[0], arg[1], arg[2]
local receivers = { select(3, unpack(arg)) }
assert(outPath and giverPath and receivers[1], "usage: poe2_party.lua <out.json> <giver.xml> <receiver.xml>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function read(path)
	local f = assert(io.open(path))
	local text = f:read("*a")
	f:close()
	return text
end

local function exact(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return tostring(v)
	end
	return setmetatable({}, { __tojson = function()
		return string.format("%.17g", v)
	end })
end

local function numbers(t)
	local out = {}
	for k, v in pairs(t) do
		if type(v) == "number" then
			out[k] = exact(v)
		end
	end
	return out
end

local function recalc()
	build.buildFlag = true
	runCallback("OnFrame")
end

local exported, order = {}, {}
if giverPath:match("%.txt$") then
	local name
	for line in (read(giverPath) .. "\n"):gmatch("([^\n]*)\n") do
		local header = line:match("^=== (.+)$")
		if header then
			name = header
			order[#order + 1] = name
		elseif name then
			exported[name] = exported[name] and (exported[name] .. "\n" .. line) or line
		end
	end
	for _, n in ipairs(order) do
		exported[n] = exported[n]:gsub("\n+$", "")
	end
else
	loadBuildFromXML(read(giverPath))
	build.partyTab.enableExportBuffs = true
	recalc()
	local saved = {}
	build.partyTab:Save(saved)
	for _, node in ipairs(saved) do
		if node.elem == "ExportedBuffs" then
			exported[node.attrib.name] = node[1]
			order[#order + 1] = node.attrib.name
		end
	end
end

local out = { exported = exported, exportOrder = order, receivers = {} }
for _, path in ipairs(receivers) do
	loadBuildFromXML(read(path))
	recalc()
	local env = build.calcsTab.mainEnv
	local without = { player = numbers(env.player.output), enemy = numbers(env.enemy.output) }
	local xml = { attrib = {} }
	for _, name in ipairs(order) do
		xml[#xml + 1] = { elem = "ImportedBuffs", attrib = { name = name }, exported[name] }
	end
	build.partyTab:Load(xml)
	recalc()
	env = build.calcsTab.mainEnv
	out.receivers[#out.receivers + 1] = {
		build = path:match("([^/\\]+)$"),
		without = without,
		with = { player = numbers(env.player.output), enemy = numbers(env.enemy.output) },
	}
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode(stable(out), { indent = true }))
f:write("\n")
f:close()
