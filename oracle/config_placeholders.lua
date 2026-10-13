local selfPath = arg[0]
local listPath, outPath = arg[1], arg[2]
assert(listPath and outPath, "usage: config_placeholders.lua <list> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local out = {}
for path in io.lines(listPath) do
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	local entry = {}
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(text, path:match("[^/]+$"))
		runCallback("OnFrame")
		local configTab = build.configTab
		for k, v in pairs(configTab.configSets[configTab.activeConfigSetId].placeholder) do
			if type(v) == "number" then
				entry[k] = v
			end
		end
	end)
	if not ok then
		entry = { error = tostring(err) }
	end
	out[path:match("[^/]+$")] = entry
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode(stable(out), { indent = true }))
f:close()
io.stderr:write("config_placeholders: wrote " .. outPath .. "\n")
