-- Loads every <Item> of each build XML the way ItemsTab:Load does and prints one JSON row per item: its id and the
-- text Item:BuildRaw writes for it, which is what ItemsTab:Save puts in the build file.
-- usage: item_build_raw_oracle.lua <out.jsonl> <xml>...
local outPath = arg[1]
assert(outPath and arg[2], "usage: item_build_raw_oracle.lua <out.jsonl> <xml>...")
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
			if item.base then
				out:write('{"build":', encode(input:match("([^/]+)$")), ',"id":', encode(item.id), ',"raw":', encode(item:BuildRaw()), "}\n")
			end
		end
	end
end
out:close()
