-- Crafts items through the Items tab's Craft Item popup, then picks affixes with its affix drop-downs and roll sliders,
-- one step at a time. Prints one JSON row per step: the item as BuildRaw writes it and every shown affix control (its
-- label, list, selection and slider). Then rows per pasted item (pastes.txt, separated by =====): the item as
-- BuildRaw writes it after the paste, and after Craft.
-- usage: item_affix_oracle.lua <out.jsonl> <pastes.txt>
local outPath, pastesPath = arg[1], arg[2]
assert(outPath and pastesPath, "usage: item_affix_oracle.lua <out.jsonl> <pastes.txt>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeNumber(v)
	return string.format("%.17g", v)
end
local function encodeStrings(list)
	local parts = {}
	for i, s in ipairs(list) do
		parts[i] = encodeString(s)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

newBuild()
local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
local out = assert(io.open(outPath, "w"))

local function controlsJSON()
	local item = tab.displayItem
	local parts = {}
	for i = 1, item.affixLimit do
		local drop = tab.controls["displayItemAffix" .. i]
		local list = {}
		for j, entry in ipairs(drop.list) do
			if type(entry) == "string" then
				list[j] = '{"label":' .. encodeString(entry) .. '}'
			else
				list[j] = '{"label":' .. encodeString(entry.label) .. ',"modList":' .. encodeStrings(entry.modList) ..
					(entry.modId and (',"modId":' .. encodeString(entry.modId)) or "") ..
					',"haveRange":' .. tostring(entry.haveRange and true or false) .. '}'
			end
		end
		parts[i] = '{"outputTable":' .. encodeString(drop.outputTable) .. ',"outputIndex":' .. drop.outputIndex ..
			',"selIndex":' .. drop.selIndex .. ',"sliderShown":' .. tostring(drop.slider.shown == true) ..
			',"sliderVal":' .. encodeNumber(drop.slider.val) .. ',"divCount":' .. encodeNumber(drop.slider.divCount or 0) ..
			',"list":[' .. table.concat(list, ",") .. ']}'
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local function row(case, step)
	out:write('{"case":', encodeString(case), ',"step":', encodeString(step), ',"raw":',
		encodeString(tab.displayItem:BuildRaw()), ',"affixLimit":', tab.displayItem.affixLimit, ',"controls":',
		controlsJSON(), "}\n")
end

-- { rarity, base, { { affix control, list entry, slider value }, ... } }
local cases = {
	{ 3, "Ruby Ring", { { 1, 2, 0.3 }, { 4, 5, 0.75 }, { 2, 4, 0 }, { 5, 3, 1 }, { 1, 3, 0.5 } } },
	{ 3, "Rusted Cuirass", { { 1, 6, 0.2 }, { 2, 3, 0.9 }, { 3, 2, 0.4 }, { 4, 7, 0.6 }, { 6, 2, 0.1 } } },
	{ 2, "Crude Bow", { { 1, 3, 0.45 }, { 2, 2, 0.55 } } },
	{ 2, "Ultimate Life Flask", { { 1, 2, 0.5 }, { 2, 3, 0.25 } } },
	{ 2, "Golden Charm", { { 1, 2, 0.8 }, { 2, 2, 0.2 } } },
	{ 3, "Ruby", { { 1, 2, 0.35 }, { 3, 4, 0.65 } } },
	{ 3, "Time-Lost Ruby", { { 1, 2, 0.5 }, { 3, 2, 0.5 } } },
	{ 3, "Absent Amulet", { { 1, 3, 0.15 }, { 2, 4, 0.95 }, { 4, 2, 0.5 } } },
}
for _, c in ipairs(cases) do
	local rarity, baseName, steps = c[1], c[2], c[3]
	local base = data.itemBases[baseName]
	assert(base and not base.hidden, "REFUSING: no shown base " .. baseName)
	local typ = base.type .. (base.subType and (": " .. base.subType) or "")
	tab:CraftItem()
	local controls = main.popups[1].controls
	controls.rarity.selIndex = rarity
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
	local case = rarity .. " " .. baseName
	row(case, "craft")
	for _, s in ipairs(steps) do
		local control, entry, val = s[1], s[2], s[3]
		local drop = tab.controls["displayItemAffix" .. control]
		assert(control <= tab.displayItem.affixLimit, "REFUSING: " .. case .. " has no affix control " .. control)
		if entry <= #drop.list then
			drop.slider.val = val
			drop.selIndex = entry
			drop.selFunc(entry, drop.list[entry])
			row(case, "select " .. control .. " " .. entry .. " " .. val)
			if drop.slider.shown then
				drop.slider.val = 1 - val
				drop.slider.changeFunc(drop.slider.val)
				tab:UpdateAffixControls()
				row(case, "slide " .. control .. " " .. (1 - val))
			end
		end
	end
	tab:SetDisplayItem()
end

local function readFile(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end
local pastes = {}
for part in (readFile(pastesPath):gsub("\n$", "") .. "\n=====\n"):gmatch("(.-)\n=====\n") do
	pastes[#pastes + 1] = part
end
for i, raw in ipairs(pastes) do
	local ok, item = pcall(function() return new("Item"):Item(raw) end)
	assert(ok, "REFUSING: paste " .. i .. ": " .. tostring(item))
	item:NormaliseQuality()
	out:write('{"case":"paste ', i, '","step":"paste","raw":', encodeString(item:BuildRaw()), ',"affixLimit":',
		item.affixLimit, ',"controls":[]}\n')
	if item.crafted then
		item:Craft()
		out:write('{"case":"paste ', i, '","step":"craft","raw":', encodeString(item:BuildRaw()), ',"affixLimit":',
			item.affixLimit, ',"controls":[]}\n')
	end
end
out:close()
