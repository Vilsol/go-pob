-- Loads a build and builds the Items tab's tooltip for a few items, one JSON row per item: the tooltip's lines from the
-- first stat difference header on, and every calculation the tooltip ran (its override and the whole numeric output,
-- the minion's too), after a row with the calculator's base pass and the main skill's flags.
-- usage: item_compare_oracle.lua <out.jsonl> <build.xml> <pastes.txt>
local outPath, xmlPath, pastesPath = arg[1], arg[2], arg[3]
assert(outPath and xmlPath and pastesPath, "usage: item_compare_oracle.lua <out.jsonl> <build.xml> <pastes.txt>")

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
local function encodeOutput(output)
	local keys = {}
	for k, v in pairs(output or {}) do
		if type(v) == "number" then
			keys[#keys + 1] = k
		end
	end
	table.sort(keys)
	local parts = {}
	for i, k in ipairs(keys) do
		parts[i] = encodeString(k) .. ":" .. encodeNumber(output[k])
	end
	return "{" .. table.concat(parts, ",") .. "}"
end
local function encodeFlags(flags)
	local keys = {}
	for k, v in pairs(flags or {}) do
		if v == true then
			keys[#keys + 1] = encodeString(k) .. ":true"
		end
	end
	table.sort(keys)
	return "{" .. table.concat(keys, ",") .. "}"
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
loadBuildFromXML(readFile(xmlPath), "item_compare")
local build = __mainObject__.main.modes.BUILD
local tab = build.itemsTab
build.buildFlag = true
runCallback("OnFrame")

-- record every calculation the tooltip runs
local calls
local getMiscCalculator = build.calcsTab.GetMiscCalculator
build.calcsTab.GetMiscCalculator = function(...)
	local calcFunc, calcBase = getMiscCalculator(...)
	return function(override, ...)
		local output = calcFunc(override, ...)
		calls[#calls + 1] = { override = override, output = output }
		return output
	end, calcBase
end

local out = assert(io.open(outPath, "w"))
local env = build.calcsTab.mainEnv
local _, calcBase = getMiscCalculator(build.calcsTab)
out:write('{"base":', encodeOutput(calcBase), ',"baseMinion":', encodeOutput(calcBase.Minion), ',"skillFlags":',
	encodeFlags(env.player.mainSkill.activeEffect.statSet.skillFlags), ',"minion":', tostring(env.player.mainSkill.minion ~= nil), "}\n")

local function compare(label, item)
	calls = {}
	local tooltip = new("Tooltip"):Tooltip()
	tab:AddItemTooltip(tooltip, item)
	local lines, on = {}, false
	for _, line in ipairs(tooltip.lines) do
		local text = line.text or ""
		if text:match("will give you:") or text:match("^%^7Player:") then
			on = true
		end
		if on and text ~= "" then
			lines[#lines + 1] = encodeString(text)
		end
	end
	local calcParts = {}
	for i, c in ipairs(calls) do
		local o = c.override or {}
		calcParts[i] = '{"slot":' .. encodeString(o.repSlotName or "") .. ',"repItem":' .. tostring(o.repItem ~= nil) ..
			',"toggle":' .. tostring((o.toggleFlask or o.toggleCharm) ~= nil) .. ',"output":' .. encodeOutput(c.output) ..
			',"minion":' .. encodeOutput(c.output.Minion) .. "}"
	end
	out:write('{"item":', encodeString(label), ',"lines":[', table.concat(lines, ","), '],"calcs":[', table.concat(calcParts, ","), "]}\n")
end

local function pasted(i)
	local item = new("Item"):Item(pastes[i])
	item:NormaliseQuality()
	item:BuildModList()
	return item
end
compare("paste helmet", pasted(1))
compare("paste wand", pasted(2))
compare("paste flask", pasted(4))
compare("equipped helmet", tab.items[3])
compare("Andvarius", main.uniqueDB.list["Andvarius, Gold Ring"])
compare("Headhunter", main.uniqueDB.list["Headhunter, Heavy Belt"])
compare("Arakaali's Gift", main.uniqueDB.list["Arakaali's Gift, Antidote Charm"])
out:close()
