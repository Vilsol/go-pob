-- usage (from PoB-PoE2's src/): luajit poe2_compare_power.lua <primary.xml> <compared.xml> <out.json> <stat>...
-- Records CompareTab:ComparePowerBuilder's rows for the compared build against the primary, per power stat, every
-- category on: category, name, config var, node id, slot and the impact as the report shows it (scaled by the display stat).
local selfPath, primaryPath, comparedPath, outPath = arg[0], arg[1], arg[2], arg[3]
assert(primaryPath and comparedPath and outPath and arg[4], "usage: poe2_compare_power.lua <primary.xml> <compared.xml> <out.json> <stat>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function read(path)
	local f = assert(io.open(path))
	local text = f:read("*a")
	f:close()
	return text
end

loadBuildFromXML(read(primaryPath))
runCallback("OnFrame")

local tab = build.compareTab
assert(tab:ImportBuild(read(comparedPath), "Compared"), "the compared build did not import")
local entry = tab.compareEntries[tab.activeCompareIndex]

local out = {}

-- the Items view's rows (CompareTab.lua:3670-3675, 3856-3907) and the Equip button's comparison (3936-3966), as
-- output differences of the power stats
local items, equip = {}, {}
local baseSlots = { "Weapon 1", "Weapon 2", "Weapon 1 Swap", "Weapon 2 Swap", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Belt", "Flask 1", "Flask 2", "Charm 1", "Charm 2", "Charm 3" }
if tab:ShouldShowRing3(entry) then
	table.insert(baseSlots, 12, "Ring 3")
end
local rows = {}
for _, slotName in ipairs(baseSlots) do
	local pSlot, cSlot = build.itemsTab.slots[slotName], entry.itemsTab.slots[slotName]
	rows[#rows + 1] = { slot = slotName, label = slotName, p = pSlot and build.itemsTab.items[pSlot.selItemId], c = cSlot and entry.itemsTab.items[cSlot.selItemId] }
end
for _, j in ipairs(tab:GetJewelComparisonSlots(entry)) do
	rows[#rows + 1] = { slot = j.pSlotName, label = j.label, p = j.pItem, c = j.cItem, pWarn = (j.pItem and not j.pNodeAllocated) and true or false, cWarn = (j.cItem and not j.cNodeAllocated) and true or false }
end
local calcFunc, calcBase = tab.calcs.getMiscCalculator(build)
for _, r in ipairs(rows) do
	items[#items + 1] = { slot = r.slot, label = r.label, p = r.p and r.p.name, c = r.c and r.c.name, pWarn = r.pWarn, cWarn = r.cWarn }
	if r.c then
		local newItem = new("Item"):Item(r.c.raw)
		newItem:NormaliseQuality()
		local pSlot = build.itemsTab.slots[r.slot]
		local selItem = pSlot and build.itemsTab.items[pSlot.selItemId]
		local override = { repSlotName = r.slot, repItem = newItem }
		if pSlot and pSlot.nodeId and not build.spec.allocNodes[pSlot.nodeId] and build.spec.nodes[pSlot.nodeId] then
			override.addNodes = { [build.spec.nodes[pSlot.nodeId]] = true }
		end
		local output = calcFunc(override)
		local slotLabel = pSlot and pSlot.label or r.slot
		local header
		if selItem then
			header = string.format("^7Equipping this item in %s will give you:\n(replacing %s%s^7)", slotLabel, colorCodes[selItem.rarity] or "^7", selItem.name)
		else
			header = string.format("^7Equipping this item in %s will give you:", slotLabel)
		end
		local delta = {}
		for i = 4, #arg do
			delta[arg[i]] = (output[arg[i]] or 0) - (calcBase[arg[i]] or 0)
		end
		equip[r.slot] = { header = header, delta = delta }
	end
end
out.items = items

-- the expanded mode's lines (CompareTab.lua:3494-3659, 3803-3807), DrawString's text in order
local tradeHelpers = LoadModule("Classes/TradeHelpers")
local cards = {}
local drawString = DrawString
for _, r in ipairs(rows) do
	if r.p or r.c then
		local sides = {}
		for _, pair in ipairs({ { r.p, r.c }, { r.c, r.p } }) do
			local lines = {}
			DrawString = function(_, _, _, _, _, text)
				lines[#lines + 1] = text
			end
			tab:DrawItemExpanded(pair[1], 0, 0, 400, tradeHelpers.buildModMap(pair[2]))
			sides[#sides + 1] = lines
		end
		cards[r.slot] = { a = sides[1], b = sides[2] }
	end
end
DrawString = drawString
out.cards = cards

-- each CalcSections entry's mod rows matched between the builds' players (CompareTab.lua:4545-4561)
local calcsHelpers = LoadModule("Classes/CompareCalcsHelpers")
local modRows = {}
local pActor, cActor = build.calcsTab.calcsEnv.player, entry.calcsTab.calcsEnv.player
for _, secDef in ipairs(tab.calcSections) do
	for _, subSec in ipairs(secDef[5]) do
		for _, rowData in ipairs(subSec.data) do
			if rowData.label and rowData[1] and rowData[1].format then
				for _, sectionData in ipairs(rowData[1]) do
					if sectionData.modName then
						local primaryRows = calcsHelpers.TabulateMods(sectionData, pActor)
						local compareRows = calcsHelpers.TabulateMods(sectionData, cActor)
						local match = #primaryRows == #compareRows
						if match then
							local counts = {}
							for _, row in ipairs(primaryRows) do
								local key = calcsHelpers.ModRowKey(row) .. "|" .. tostring(row.value)
								counts[key] = (counts[key] or 0) + 1
							end
							for _, row in ipairs(compareRows) do
								local key = calcsHelpers.ModRowKey(row) .. "|" .. tostring(row.value)
								if not counts[key] then
									match = false
									break
								end
								counts[key] = counts[key] > 1 and counts[key] - 1 or nil
							end
						end
						local names = {}
						for _, name in ipairs(type(sectionData.modName) == "table" and sectionData.modName or { sectionData.modName }) do
							names[#names + 1] = name
						end
						-- FormatModRow's columns of each side's rows (CompareCalcsHelpers.lua:130-146)
						local function formatted(rows, b)
							local out = {}
							for _, row in ipairs(rows) do
								local display, sourceType, sourceName, modName = calcsHelpers.FormatModRow(row, sectionData, b)
								out[#out + 1] = { key = calcsHelpers.ModRowKey(row), value = type(row.value) == "number" and row.value or nil, display = tostring(display),
									sourceType = sourceType, sourceName = sourceName, modName = modName }
							end
							return out
						end
						modRows[#modRows + 1] = {
							pRows = formatted(primaryRows, build), cRows = formatted(compareRows, entry),
							modName = names, multi = type(sectionData.modName) == "table", modType = sectionData.modType, cfg = sectionData.cfg,
							modSource = sectionData.modSource, enemy = sectionData.enemy and true or false, actor = sectionData.actor,
							rows = #primaryRows, match = match,
						}
					end
				end
			end
		end
	end
end
out.modRows = modRows
out.equip = equip

for i = 4, #arg do
	local powerStat
	for _, s in ipairs(data.powerStatList) do
		if s.stat == arg[i] then
			powerStat = s
		end
	end
	assert(powerStat, "no power stat " .. arg[i])
	tab:ComparePowerBuilder(entry, powerStat, { treeNodes = true, items = true, skillGems = true, supportGems = true, config = true })
	local rows = {}
	for _, r in ipairs(tab.comparePowerResults) do
		local var
		if r.category == "Config" then
			for _, varData in ipairs(tab.configOptions) do
				local label = varData.label and varData.label:gsub(":$", "")
				if label and varData.apply and r.name:sub(1, #label + 3) == label .. "  (" then
					var = varData.var
				end
			end
		end
		rows[#rows + 1] = { category = r.category, name = r.name, var = var, nodeId = r.nodeId, slotName = r.slotName, impact = r.impact, pathDist = r.pathDist }
	end
	table.sort(rows, function(a, b)
		if a.category ~= b.category then
			return a.category < b.category
		end
		return a.name < b.name
	end)
	out[arg[i]] = rows
end


local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
