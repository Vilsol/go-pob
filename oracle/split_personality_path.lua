local outPath = arg[1]
local files = {}
for i = 2, #arg do
	files[#files + 1] = arg[i]
end
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local out = {}
for _, path in ipairs(files) do
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	loadBuildFromXML(xmlText, path)
	runCallback("OnFrame")
	local ids = {}
	for id in pairs(build.spec.splitPersonalityPath or {}) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	out[path:match("([^/]+)$")] = ids
	io.stderr:write(string.format("split-personality-path: %s, %d nodes\n", path:match("([^/]+)$"), #ids))
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out))
f:close()
