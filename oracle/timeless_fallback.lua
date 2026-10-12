local outPath, corpus = arg[1], arg[2]
assert(outPath and corpus, "usage: timeless_fallback.lua <out.json> <corpus dir>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local cases = {
	{ build = "0005-occultist-vaal-double-strike.xml", type = 2, stat = "Life" },
	{ build = "0005-occultist-vaal-double-strike.xml", type = 3, stat = "CombinedDPS" },
	{ build = "0028-occultist-forbidden-rite-of-soul-sacrifice.xml", type = 4, stat = "EnergyShield" },
	{ build = "0010-elementalist-storm-burst-2.xml", type = 5, stat = "CritChance" },
	{ build = "0010-elementalist-storm-burst-2.xml", type = 6, stat = "TotalEHP" },
}

local out = {}
for _, c in ipairs(cases) do
	local fh = assert(io.open(corpus .. "/" .. c.build, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	loadBuildFromXML(xmlText, c.build)
	runCallback("OnFrame")
	local td = build.timelessData
	td.jewelType = { id = c.type }
	td.conquerorType = { id = 1 }
	td.jewelSocket = { id = -1 }
	td.searchList = ""
	td.searchListFallback = ""
	build.treeTab:FindTimelessJewel()
	local controls = main.popups[1].controls
	local index
	for i, entry in ipairs(controls.fallbackWeightsList.list) do
		if entry.stat == c.stat then
			index = i
		end
	end
	assert(index, "REFUSING: no fallback weight mode for " .. c.stat)
	controls.fallbackWeightsList:SetSel(index)
	controls.fallbackWeightsButton.onClick()
	local modes = {}
	for _, entry in ipairs(controls.fallbackWeightsList.list) do
		modes[#modes + 1] = { label = entry.label, stat = entry.stat }
	end
	out[#out + 1] = { build = c.build, type = c.type, stat = c.stat, list = td.searchListFallback or "", modes = modes }
	io.stderr:write(string.format("timeless-fallback: %s type %d %s, %d lines\n", c.build, c.type, c.stat, select(2, (td.searchListFallback or ""):gsub("\n", "")) + 1))
	main:ClosePopup()
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out, { indent = true }))
f:close()
