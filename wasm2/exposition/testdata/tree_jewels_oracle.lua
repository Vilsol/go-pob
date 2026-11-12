-- Loads a build with jewels in its tree sockets and equips them through the sockets' item slots, one step at a time,
-- recalculating after each. Prints one JSON row per step: the step name, what each tree socket holds and every
-- numeric main output.
-- usage: tree_jewels_oracle.lua <out.jsonl> <build.xml>
local outPath, xmlPath = arg[1], arg[2]
assert(outPath and xmlPath, "usage: tree_jewels_oracle.lua <out.jsonl> <build.xml>")

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

local fh = assert(io.open(xmlPath, "r"))
loadBuildFromXML(fh:read("*a"), "tree_jewels")
fh:close()

local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
local out = assert(io.open(outPath, "w"))
local function row(step)
	build.buildFlag = true
	runCallback("OnFrame")
	local output = build.calcsTab.mainOutput
	local keys = {}
	for k, v in pairs(output) do
		if type(v) == "number" then
			keys[#keys + 1] = k
		end
	end
	table.sort(keys)
	local parts = {}
	for _, k in ipairs(keys) do
		parts[#parts + 1] = encodeString(k) .. ":" .. encodeNumber(output[k])
	end
	local sockets = {}
	for _, slot in ipairs(tab.orderedSlots) do
		if slot.nodeId and slot.selItemId and slot.selItemId ~= 0 then
			sockets[#sockets + 1] = encodeString(slot.slotName) .. ":" .. slot.selItemId
		end
	end
	out:write('{"step":', encodeString(step), ',"sockets":{', table.concat(sockets, ","), '},"output":{',
		table.concat(parts, ","), "}}\n")
end
-- ItemSlotControl.lua:23-30
local function equip(slotName, id)
	local slot = tab.slots[slotName]
	slot:SetSelItemId(id)
	tab:PopulateSlots()
	tab:AddUndoState()
end

row("load")
equip("Jewel 21984", 0)
row("empty socket 21984")
equip("Jewel 21984", 7)
row("Rune Spark in socket 21984")
equip("Jewel 7960", 0)
row("empty socket 7960")
out:close()
