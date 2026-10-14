local buildsDir = arg[1]
assert(buildsDir, "usage: calcs_attribute_breakdowns.lua <buildsDir>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local lfs = require("lfs")

local builds = { }
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)

local keys = { "Str", "Dex", "Int", "Omni" }

local function attributes(bd)
	local out = { }
	for _, key in ipairs(keys) do
		local b = bd[key]
		if b then
			local lines = { }
			for _, l in ipairs(b) do
				lines[#lines + 1] = l
			end
			out[key] = lines
		end
	end
	return out
end

for i, file in ipairs(builds) do
	if i % 40 == 1 then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			local bd = build.calcsTab.calcsEnv.player.breakdown
			print(json.encode({ build = file, attributes = attributes(bd) }, { keyorder = { "build", "attributes" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
