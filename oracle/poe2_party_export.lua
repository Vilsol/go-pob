-- usage (from PoB-PoE2's src/): luajit poe2_party_export.lua <out.json> <giver.xml>...
-- Records what PoB-PoE2's party tab exports for each giver build with the Import tab's "Export support" on
-- (PartyTab:Save's ExportedBuffs), keyed by the giver's file name, each with its buffers in save order.
local selfPath, outPath = arg[0], arg[1]
local givers = { select(2, unpack(arg)) }
assert(outPath and givers[1], "usage: poe2_party_export.lua <out.json> <giver.xml>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function read(path)
	local f = assert(io.open(path))
	local text = f:read("*a")
	f:close()
	return text
end

local out = {}
for _, path in ipairs(givers) do
	loadBuildFromXML(read(path))
	build.partyTab.enableExportBuffs = true
	build.buildFlag = true
	runCallback("OnFrame")
	local saved = {}
	build.partyTab:Save(saved)
	local buffers = {}
	for _, node in ipairs(saved) do
		if node.elem == "ExportedBuffs" then
			buffers[#buffers + 1] = { name = node.attrib.name, text = node[1] }
		end
	end
	out[path:match("([^/\\]+)$")] = buffers
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode(stable(out), { indent = true }))
f:write("\n")
f:close()
