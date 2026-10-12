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
	local groups = {}
	local n = #tab.socketGroupList
	for g = 1, n do
		local copied
		Copy = function(text) copied = text end
		tab:CopySocketGroup(tab.socketGroupList[g])
		local row = { text = copied }
		if copied then
			Paste = function() return nil end
			local before = #tab.socketGroupList
			tab:PasteSocketGroup(copied)
			if #tab.socketGroupList > before then
				local pasted = tab.socketGroupList[#tab.socketGroupList]
				local gems = {}
				for _, gem in ipairs(pasted.gemList) do
					gems[#gems + 1] = { nameSpec = gem.nameSpec, level = gem.level, quality = gem.quality, enabled = gem.enabled, count = gem.count }
				end
				row.pasted = { label = pasted.label, slot = pasted.slot, gems = gems }
				table.remove(tab.socketGroupList)
			end
		end
		groups[g] = row
	end
	build.modFlag = false
	print(json.encode({ file = paths[i]:match("[^/]+$"), groups = groups }))
end
