-- Drives the Items tab's Craft Item popup for a few rarity/base choices and prints one JSON row per choice: the item
-- it opens in the editor, as BuildRaw writes it. Then one row per data.itemBaseTypeList entry with its bases' names in
-- data.itemBaseLists order.
-- usage: item_craft_oracle.lua <out.jsonl>
local outPath = arg[1]
assert(outPath, "usage: item_craft_oracle.lua <out.jsonl>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end

newBuild()
local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
local out = assert(io.open(outPath, "w"))

local choices = {
	{ 3, "", "Two-Stone Ring" },
	{ 3, "Doom Grip", "Adherent Cuffs" },
	{ 2, "", "Crude Bow" },
	{ 1, "", "Ultimate Life Flask" },
	{ 3, "", "Thawing Charm" },
	{ 3, "", "Ruby" },
	{ 4, "", "Rusted Cuirass" },
	{ 1, "", "Attuned Wand" },
	{ 3, "", "Golden Charm" },
	{ 2, "", "Absent Amulet" },
}
for _, c in ipairs(choices) do
	local rarity, title, baseName = c[1], c[2], c[3]
	local base = data.itemBases[baseName]
	assert(base and not base.hidden, "REFUSING: no shown base " .. baseName)
	local typ = base.type .. (base.subType and (": " .. base.subType) or "")
	tab:CraftItem()
	local controls = main.popups[1].controls
	controls.rarity.selIndex = rarity
	controls.title:SetText(title)
	for i, t in ipairs(data.itemBaseTypeList) do
		if t == typ then
			controls.type.selIndex = i
			controls.type.selFunc(i, t)
		end
	end
	for i, b in ipairs(controls.base.list) do
		if b.name == baseName then
			controls.base.selIndex = i
		end
	end
	controls.save.onClick()
	out:write('{"rarity":', rarity, ',"title":', encodeString(title), ',"base":', encodeString(baseName), ',"raw":',
		encodeString(tab.displayItem:BuildRaw()), "}\n")
	tab:SetDisplayItem()
end

for _, typ in ipairs(data.itemBaseTypeList) do
	local names = {}
	for _, b in ipairs(data.itemBaseLists[typ]) do
		names[#names + 1] = encodeString(b.name) .. "," .. encodeString(b.label)
	end
	out:write('{"type":', encodeString(typ), ',"bases":[', table.concat(names, ","), "]}\n")
end
out:close()
