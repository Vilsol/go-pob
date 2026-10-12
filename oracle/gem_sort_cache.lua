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
	local tab = build.skillsTab
	for g, group in ipairs(tab.socketGroupList) do
		tab:SetDisplayGroup(group)
		local control = tab.gemSlots[1].nameSpec
		control.sortCache = nil
		control:UpdateSortCache()
		local pending = { }
		for _, gemId in ipairs(control.sortCache.pendingGems or { }) do
			pending[gemId] = true
		end
		local canSupport, pendingIds, universe = { }, { }, { }
		for gemId, gemData in pairs(control.gems) do
			if control.sortCache.canSupport[gemId] then
				canSupport[#canSupport + 1] = gemData.grantedEffectId
			end
			if pending[gemId] then
				pendingIds[#pendingIds + 1] = gemData.grantedEffectId
			end
			universe[#universe + 1] = { gemData.gameId or gemData.id, gemData.grantedEffectId, gemData.variantId or "" }
		end
		table.sort(canSupport)
		table.sort(pendingIds)
		if g == 1 then
			table.sort(universe, function(a, b) return a[2] < b[2] end)
			print(json.encode({ file = paths[i]:match("[^/]+$"), gems = universe }))
		end
		print(json.encode({ file = paths[i]:match("[^/]+$"), group = g - 1, canSupport = canSupport, pending = pendingIds }))
	end
end
