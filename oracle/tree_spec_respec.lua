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
	local tab = build.treeTab
	build.calcsTab.GetMiscCalculator = function() return nil end
	for s = 1, #tab.specList do
		if s ~= tab.activeSpec then
			local lines = { }
			local tooltip = { Clear = function() end, AddLine = function(_, _, text) lines[#lines + 1] = text end }
			tab.controls.specSelect.tooltipFunc(tooltip, "HOVER", s)
			local respec = { }
			for _, l in ipairs(lines) do
				if l:find("Gold") or l:find("refunded") then
					respec[#respec + 1] = l
				end
			end
			print(json.encode({ file = paths[i]:match("[^/]+$"), spec = s - 1, level = build.characterLevel, lines = respec }))
		end
	end
end
