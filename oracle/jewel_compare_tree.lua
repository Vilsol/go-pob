local outPath, corpus = arg[1], arg[2]
assert(outPath and corpus, "usage: jewel_compare_tree.lua <out.json> <corpus dir>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local STATS = { "Life", "Mana", "EnergyShield", "Str", "Dex", "Int", "TotalEHP", "CombinedDPS", "Armour", "Evasion" }

local cases = {
	{ build = "0005-occultist-vaal-double-strike.xml", jewel = "socketed:Lethal Pride" },
	{ build = "0005-occultist-vaal-double-strike.xml", jewel = "Lethal Pride\nTimeless Jewel\nLeague: Legion\nLimited to: 1 Historic\nVariant: Kaom (Strength of Blood)\nVariant: Rakiata (Tempered by War)\nVariant: Akoya (Chainbreaker)\nSelected Variant: 2\nRadius: Large\nImplicits: 0\n{variant:1}Commanded leadership over 17288 warriors under Kaom\n{variant:2}Commanded leadership over 17288 warriors under Rakiata\n{variant:3}Commanded leadership over 17288 warriors under Akoya\nPassives in radius are Conquered by the Karui\nHistoric\n" },
	{ build = "0007-elementalist-storm-burst-of-repulsion.xml", jewel = "socketed:Glorious Vanity" },
	{ build = "0028-occultist-forbidden-rite-of-soul-sacrifice.xml", jewel = "socketed:Militant Faith" },
}

local function load(file)
	local fh = assert(io.open(corpus .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	loadBuildFromXML(xmlText, file)
	runCallback("OnFrame")
end

local out = {}
for _, c in ipairs(cases) do
	load(c.build)
	local item
	local name = c.jewel:match("^socketed:(.+)$")
	if name then
		for nodeId, itemId in pairs(build.spec.jewels) do
			local it = build.itemsTab.items[itemId]
			if it and it.name:find(name, 1, true) and build.spec.allocNodes[nodeId] then
				item = it
			end
		end
		assert(item, "REFUSING: " .. c.build .. " has no socketed " .. name)
	else
		item = new("Item", c.jewel)
	end
	local calls = {}
	local getMisc = build.calcsTab.GetMiscCalculator
	build.calcsTab.GetMiscCalculator = function(tab, ...)
		local calcFunc, calcBase = getMisc(tab, ...)
		return function(override, ...)
			local output = calcFunc(override, ...)
			if override and override.repSlotName and override.repSlotName:match("^Jewel ") then
				local stats = {}
				for _, s in ipairs(STATS) do
					stats[s] = output[s] or 0
				end
				calls[#calls + 1] = {
					slot = override.repSlotName,
					removing = override.repItem == nil,
					rebuilt = override.spec ~= nil,
					stats = stats,
				}
			end
			return output
		end, calcBase
	end
	local tooltip = new("Tooltip")
	build.itemsTab:AddItemTooltip(tooltip, item)
	build.calcsTab.GetMiscCalculator = getMisc
	local base = {}
	local _, calcBase = build.calcsTab:GetMiscCalculator()
	for _, s in ipairs(STATS) do
		base[s] = calcBase[s] or 0
	end
	table.sort(calls, function(a, b) return a.slot < b.slot end)
	out[#out + 1] = { build = c.build, item = item.raw, socketed = name ~= nil, base = base, calls = calls }
	io.stderr:write(string.format("jewel-compare-tree: %s %s, %d sockets\n", c.build, name or "new jewel", #calls))
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out, { indent = true }))
f:close()
