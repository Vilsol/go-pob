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
	for g, group in ipairs(build.skillsTab.socketGroupList) do
		local lines = { }
		local tooltip = {
			AddLine = function(_, _, text)
				for line in (text .. "\n"):gmatch("([^\n]*)\n") do
					lines[#lines + 1] = line
				end
			end,
			AddSeparator = function()
				if lines[#lines] ~= "----" then
					lines[#lines + 1] = "----"
				end
			end,
		}
		build.skillsTab:AddSocketGroupTooltip(tooltip, group)
		print(json.encode({ file = paths[i]:match("[^/]+$"), group = g - 1, lines = lines }))
	end
end
