local outPath = arg[1]
assert(outPath, "usage: process_gem_level.lua <out.json>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local modes = { "normalMaximum", "corruptedMaximum", "awakenedMaximum", "characterLevel", "levelOne" }
local levels = { 1, 28, 70, 100 }
local out = {}
local n = 0
local seen = {}
for gemId, gemData in pairs(build.data.gems) do
	local key = gemData.gameId or gemId
	seen[key] = (seen[key] or 0) + 1
end
for gemId, gemData in pairs(build.data.gems) do
	if seen[gemData.gameId or gemId] > 1 then
		goto continue
	end
	local row = {}
	for _, mode in ipairs(modes) do
		build.skillsTab.defaultGemLevel = mode
		for _, level in ipairs(levels) do
			build.characterLevel = level
			row[mode .. ":" .. level] = build.skillsTab:ProcessGemLevel(gemData)
		end
	end
	out[gemData.gameId or gemId] = row
	n = n + 1
	::continue::
end
assert(n > 0, "REFUSING: data.gems is empty")
local file = assert(io.open(outPath, "w"))
file:write(json.encode(out))
file:close()
io.stderr:write(string.format("wrote %s (%d gems)\n", outPath, n))
