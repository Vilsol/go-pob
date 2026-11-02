-- Loads a build and edits its items through the Items tab's own controls, one step at a time, recalculating after
-- each. Prints one JSON row per step: the step name, the main output stats, the active item set and what each shown
-- slot holds, then the saved <Items> and <Tree> elements after the last step.
-- usage: items_edit_oracle.lua <out.jsonl> <build.xml> <pastes.txt>
local outPath, xmlPath, pastesPath = arg[1], arg[2], arg[3]
assert(outPath and xmlPath and pastesPath, "usage: items_edit_oracle.lua <out.jsonl> <build.xml> <pastes.txt>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeNumber(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return '"' .. tostring(v) .. '"'
	end
	return string.format("%.17g", v)
end

local function readFile(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end
local pastes = {}
for part in (readFile(pastesPath) .. "\n=====\n"):gmatch("(.-)\n=====\n") do
	pastes[#pastes + 1] = part
end
loadBuildFromXML(readFile(xmlPath), "items_edit")

local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
local sets = new("ItemSetService"):ItemSetService(tab)

local stats = { "Life", "Mana", "EnergyShield", "Armour", "Evasion", "FireResist", "ColdResist", "LightningResist", "Str",
	"Dex", "Int", "Spirit", "TotalDPS", "CombinedDPS", "AverageDamage", "Speed" }
local out = assert(io.open(outPath, "w"))
local function row(step)
	build.buildFlag = true
	runCallback("OnFrame")
	local output = build.calcsTab.mainOutput
	local parts = {}
	for _, stat in ipairs(stats) do
		local v = output[stat]
		if type(v) == "number" then
			parts[#parts + 1] = encodeString(stat) .. ":" .. encodeNumber(v)
		end
	end
	local slots = {}
	for _, slot in ipairs(tab.orderedSlots) do
		if slot.selItemId and slot.selItemId ~= 0 then
			slots[#slots + 1] = encodeString(slot.slotName) .. ":" .. slot.selItemId
		end
	end
	out:write('{"step":', encodeString(step), ',"activeItemSet":', tab.activeItemSetId, ',"useSecondWeaponSet":',
		tostring(tab.activeItemSet.useSecondWeaponSet == true), ',"slots":{', table.concat(slots, ","), '},"output":{',
		table.concat(parts, ","), "}}\n")
end

local function paste(i)
	tab:CreateDisplayItemFromRaw(pastes[i], true)
	tab:AddDisplayItem()
end
-- ItemSlotControl.lua:23-30
local function equip(slotName, id)
	local slot = tab.slots[slotName]
	slot:SetSelItemId(id)
	tab:PopulateSlots()
	tab:AddUndoState()
end
-- ItemListControl.lua:321-328
local function edit(id)
	tab.controls.itemList:OnSelClick(0, id, true)
end

row("load")
paste(1)
row("paste helmet")
paste(2)
row("paste wand")
paste(3)
row("paste second wand")
-- ItemsTab.lua:358-364
tab.controls.weaponSwap2.onClick()
row("weapon set II")
equip("Weapon 1 Swap", 6)
row("equip second wand in set II")
tab.controls.weaponSwap1.onClick()
row("weapon set I")

-- ItemsTab.lua:613-624, 792-800
edit(1)
tab.controls.displayItemSocketRuneEdit.changeFunc("2")
tab.controls.displayItemRune1.selFunc(2, { name = "Desert Rune" })
tab.controls.displayItemRune2.selFunc(2, { name = "Glacial Rune" })
tab:AddDisplayItem()
row("body armour runes")
edit(5)
tab.controls.displayItemRune1.selFunc(2, { name = "Storm Rune" })
tab:AddDisplayItem()
row("wand rune")

-- ItemsTab.lua:269-273
do
	local runeSlot = tab.runeSlots["Helmet Rune #1"]
	runeSlot:SelByValue("Desert Rune", "name")
	runeSlot.selFunc(runeSlot.selIndex, runeSlot:GetSelValue())
end
row("character rune")

paste(4)
row("paste flask")
-- ItemSlotControl.lua:58-64
tab.slots["Flask 1"].controls.activate.changeFunc(true)
row("flask active")

-- ItemSetService.lua
sets:NewItemSet("Second")
row("new item set")
equip("Body Armour", 1)
row("equip body armour in new set")
sets:CopyItemSet(tab.activeItemSetId, "Third")
row("copy item set")
sets:RenameItemSet(tab.activeItemSetId, "Renamed")
row("rename item set")
-- ItemsTab.lua:161-164
tab:SetActiveItemSet(1)
tab:AddUndoState()
row("item set 1")
sets:DeleteItemSet(4, 4)
row("delete item set")
tab:SetActiveItemSet(2)
tab:AddUndoState()
row("item set 2")
tab:SetActiveItemSet(1)
tab:AddUndoState()
row("back to item set 1")

tab:DeleteItem(tab.items[2])
row("delete ring")
equip("Helmet", 0)
row("unequip helmet")

-- ItemsTab.lua:2871-2899
edit(5)
do
	local id = tab.displayItem.id
	tab:CreateDisplayItemFromRaw(pastes[5], not tab.displayItem)
	tab.displayItem.id = id
end
tab:AddDisplayItem()
row("edit wand text")

local items = { elem = "Items" }
tab:Save(items)
local tree = { elem = "Tree" }
build.treeTab:Save(tree)
local xmlOut = common.xml.ComposeXML({ elem = "PathOfBuilding2", items, tree })
out:write('{"step":"saved","xml":', encodeString(xmlOut), "}\n")
out:close()
