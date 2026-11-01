-- Loads every <Item> of each build XML the way ItemsTab:Load does and prints one JSON row per item:
-- the parsed fields, every mod line, the slot mod lists and each modLib.parseMod call it made.
-- usage: item_oracle.lua <out.jsonl> <xml>...
local outPath = arg[1]
assert(outPath and arg[2], "usage: item_oracle.lua <out.jsonl> <xml>...")
-- Read before HeadlessWrapper runs: the launch can leave #arg at 0.
local inputs = {}
while arg[#inputs + 2] do
	inputs[#inputs + 1] = arg[#inputs + 2]
end

dofile("HeadlessWrapper.lua")

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys, function(a, b)
		if type(a) == type(b) then
			return a < b
		end
		return type(a) == "number"
	end)
	return keys
end

local encode
local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
function encode(v)
	local ty = type(v)
	if ty == "number" then
		if v ~= v or v == math.huge or v == -math.huge then
			return '"' .. tostring(v) .. '"'
		end
		return string.format("%.17g", v)
	elseif ty == "string" then
		return encodeString(v)
	elseif ty == "boolean" then
		return tostring(v)
	elseif ty == "table" then
		local n = 0
		for _ in pairs(v) do
			n = n + 1
		end
		local parts = {}
		if n > 0 and n == #v then
			for i = 1, #v do
				parts[i] = encode(v[i])
			end
			return "[" .. table.concat(parts, ",") .. "]"
		end
		for _, k in ipairs(sortedKeys(v)) do
			parts[#parts + 1] = encodeString(tostring(k)) .. ":" .. encode(v[k])
		end
		return "{" .. table.concat(parts, ",") .. "}"
	elseif ty == "nil" then
		return "null"
	end
	return '"<' .. ty .. '>"'
end

local function encodeMod(m)
	local tags = {}
	for i = 1, #m do
		tags[i] = encode(m[i])
	end
	return '{"name":' .. encode(m.name) .. ',"type":' .. encode(m.type) .. ',"value":' .. encode(m.value)
		.. ',"flags":' .. encode(m.flags) .. ',"keywordFlags":' .. encode(m.keywordFlags)
		.. ',"source":' .. encode(m.source) .. ',"tags":[' .. table.concat(tags, ",") .. ']}'
end

local function encodeModList(list)
	if not list then
		return "null"
	end
	local parts = {}
	for i, m in ipairs(list) do
		parts[i] = encodeMod(m)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local parseLog
local origParseMod = modLib.parseMod
modLib.parseMod = function(line, isComb)
	local modList, extra = origParseMod(line, isComb)
	if parseLog and not parseLog[line] then
		parseLog[line] = '{"modList":' .. encodeModList(modList) .. ',"extra":' .. encode(extra) .. '}'
	end
	return modList, extra
end

local lineFlags = { "crafted", "custom", "disabled", "enchant", "fractured", "implicit", "desecrated", "mutated", "rune",
	"unscalable", "prefix", "suffix", "bonded", "socketedRuneEffectAlreadyApplied" }
-- the text formatModLine shows, colour codes stripped; nil for a hidden zero-value line
local function rangedLine(ml)
	local shown = itemLib.formatModLine(ml, false, true)
	return shown and (shown:gsub("%^x%x%x%x%x%x%x", ""):gsub("%^%d", ""))
end
local function encodeModLines(lines)
	local parts = {}
	for i, ml in ipairs(lines) do
		local flags = {}
		for _, f in ipairs(lineFlags) do
			if ml[f] then
				flags[#flags + 1] = f
			end
		end
		parts[i] = '{"line":' .. encode(ml.line) .. ',"extra":' .. encode(ml.extra) .. ',"range":' .. encode(ml.range)
			.. ',"valueScalar":' .. encode(ml.valueScalar) .. ',"flags":' .. encode(flags)
			.. ',"augmentType":' .. encode(ml.augmentType) .. ',"ranged":' .. encode(rangedLine(ml))
			.. ',"modList":' .. encodeModList(ml.modList) .. '}'
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local out = assert(io.open(outPath, "w"))
for _, input in ipairs(inputs) do
	local f = assert(io.open(input, "r"))
	local text = f:read("*a")
	f:close()
	local dbXML = assert(common.xml.ParseXML(text))
	local itemsNode
	for _, node in ipairs(dbXML[1]) do
		if type(node) == "table" and node.elem == "Items" then
			itemsNode = node
		end
	end
	for _, node in ipairs(itemsNode or {}) do
		if type(node) == "table" and node.elem == "Item" then
			-- ItemsTab.lua:1227-1281, keeping what each step was given
			parseLog = {}
			local item = new("Item"):Item("")
			item.id = tonumber(node.attrib.id)
			item.variant = tonumber(node.attrib.variant)
			local attrib = {}
			for _, k in ipairs({ "variant", "variantAlt", "variantAlt2", "variantAlt3", "variantAlt4", "variantAlt5" }) do
				attrib[k] = tonumber(node.attrib[k])
			end
			for i, k in ipairs({ "variantAlt", "variantAlt2", "variantAlt3", "variantAlt4", "variantAlt5" }) do
				if node.attrib[k] then
					item["hasAltVariant" .. (i == 1 and "" or i)] = true
					item[k] = tonumber(node.attrib[k])
				end
			end
			local raws, ranges = {}, {}
			for _, child in ipairs(node) do
				if type(child) == "string" then
					raws[#raws + 1] = child
					item:ParseRaw(child)
				elseif child.elem == "ModRange" then
					local id = tonumber(child.attrib.id) or 0
					local range = tonumber(child.attrib.range) or 1
					ranges[#ranges + 1] = { id = id, range = range }
					for _, list in ipairs{item.buffModLines, item.enchantModLines, item.implicitModLines, item.explicitModLines } do
						if id <= #list then
							list[id].range = range
							break
						end
						id = id - #list
					end
				end
			end
			local fallBackJewelSocketCount = {
				["Tabula Rasa"] = 6,
				["Sekhema's Resolve"] = 1,
			}
			if item.base then
				if fallBackJewelSocketCount[item.title] and item.jewelSocketCount == 0 then
					item.jewelSocketCount = fallBackJewelSocketCount[item.title]
				end
				item:BuildModList()
			end
			local slotMods = {}
			if item.base then
				if item.slotModList then
					for i, list in ipairs(item.slotModList) do
						slotMods[i] = encodeModList(list)
					end
				else
					slotMods[1] = encodeModList(item.modList)
				end
			end
			local logParts = {}
			for _, k in ipairs(sortedKeys(parseLog)) do
				logParts[#logParts + 1] = encodeString(k) .. ":" .. parseLog[k]
			end
			parseLog = nil
			out:write('{"build":', encode(input:match("([^/]+)$")), ',"id":', encode(item.id),
				',"raws":', encode(raws), ',"attrib":', encode(attrib), ',"ranges":', encode(ranges),
				',"name":', encode(item.name), ',"baseName":', encode(item.baseName), ',"rarity":', encode(item.rarity),
				',"title":', encode(item.title), ',"type":', encode(item.type), ',"quality":', encode(item.quality),
				',"itemLevel":', encode(item.itemLevel), ',"corrupted":', encode(item.corrupted or false),
				',"itemSocketCount":', encode(item.itemSocketCount), ',"jewelSocketCount":', encode(item.jewelSocketCount),
				',"runes":', encode(item.runes), ',"requirements":', encode(item.requirements),
				',"armourData":', encode(item.armourData), ',"weaponData":', encode(item.weaponData),
				',"flaskData":', encode(item.flaskData), ',"charmData":', encode(item.charmData),
				',"spiritValue":', encode(item.spiritValue), ',"charmLimit":', encode(item.charmLimit),
				',"buffModLines":', encodeModLines(item.buffModLines or {}),
				',"enchantModLines":', encodeModLines(item.enchantModLines or {}),
				',"runeModLines":', encodeModLines(item.runeModLines or {}),
				',"implicitModLines":', encodeModLines(item.implicitModLines or {}),
				',"explicitModLines":', encodeModLines(item.explicitModLines or {}),
				',"slotMods":[', table.concat(slotMods, ","), ']',
				',"parseLog":{', table.concat(logParts, ","), '}}\n')
		end
	end
end
out:close()
