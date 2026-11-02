-- Opens items in the Items tab's editor and moves their modifier range sliders and quality, one step at a time. Prints
-- one JSON row per step: the item as BuildRaw writes it, its range lines with their rolls, and its quality.
-- usage: item_range_oracle.lua <out.jsonl> <pastes.txt>
local outPath, pastesPath = arg[1], arg[2]
assert(outPath and pastesPath, "usage: item_range_oracle.lua <out.jsonl> <pastes.txt>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function readFile(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end

newBuild()
local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
local out = assert(io.open(outPath, "w"))

local function row(case, step)
	local item = tab.displayItem
	local lines = {}
	for i, modLine in ipairs(item.rangeLineList) do
		lines[i] = '{"line":' .. encodeString(modLine.line) .. ',"range":' .. string.format("%.17g", modLine.range or 0.5) .. '}'
	end
	out:write('{"case":', encodeString(case), ',"step":', encodeString(step), ',"raw":', encodeString(item:BuildRaw()),
		',"quality":', item.quality and tostring(item.quality) or "null", ',"qualityShown":',
		tostring(tab.controls.displayItemQualityEdit:IsShown() and true or false), ',"rangeLines":[',
		table.concat(lines, ","), "]}\n")
end

local items = {}
for part in (readFile(pastesPath):gsub("\n$", "") .. "\n=====\n"):gmatch("(.-)\n=====\n") do
	items[#items + 1] = { "paste " .. (#items + 1), part }
end
for _, name in ipairs({ "Headhunter, Heavy Belt", "Fairgraves' Curse, Artillery Bow" }) do
	local item = assert(main.uniqueDB.list[name], "REFUSING: no unique " .. name)
	items[#items + 1] = { name, item.raw }
end
for _, c in ipairs(items) do
	local case, raw = c[1], c[2]
	local item = new("Item"):Item(raw)
	item:NormaliseQuality()
	tab:SetDisplayItem(item)
	row(case, "open")
	local rolls = { 0.37, 0.74, 0.11, 0.48, 0.85, 0, 1 }
	for i = 1, #tab.displayItem.rangeLineList do
		local v = rolls[(i - 1) % #rolls + 1]
		tab.displayItem.rangeLineList[i].range = v
		tab.displayItem:BuildAndParseRaw()
		row(case, "range " .. i .. " " .. v)
	end
	if tab.controls.displayItemQualityEdit:IsShown() then
		tab.controls.displayItemQualityEdit:SetText("13", true)
		row(case, "quality 13")
	end
	tab:SetDisplayItem()
end
out:close()
