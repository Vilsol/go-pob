-- usage (from PoB's src/): luajit trade_hash.lua <out.jsonl> <stride>
-- Records TradeHelpers' option and stat description matches for every <stride>-th distinct line of the item mod
-- tables, the enchantments and the unique items, ranges at their midpoint, sorted: the option entry per mod type, the
-- trade hashes, the value and whether the trade site wants it negated ("nil" when no form said).
local outPath, stride = arg[1], tonumber(arg[2] or "1")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local tradeHelpers = require("Classes.TradeHelpers")

local seen, lines = {}, {}
local function add(line)
	if type(line) == "string" and not seen[line] then
		seen[line] = true
		table.insert(lines, line)
	end
end
local function addStrings(t)
	for _, v in pairs(t) do
		if type(v) == "table" then
			addStrings(v)
		else
			add(type(v) == "string" and itemLib.applyRange(v, 0.5) or nil)
		end
	end
end
for _, mods in pairs(data.itemMods) do
	for _, mod in pairs(mods) do
		for _, line in ipairs(mod) do
			add(itemLib.applyRange(line, 0.5))
		end
	end
end
addStrings(data.enchantments)
for _, item in pairs(main.uniqueDB.list) do
	for _, list in ipairs({ item.enchantModLines, item.implicitModLines, item.explicitModLines, item.scourgeModLines }) do
		for _, modLine in ipairs(list or {}) do
			add(modLine.range and itemLib.applyRange(modLine.line, modLine.range, modLine.valueScalar) or modLine.line)
		end
	end
end
table.sort(lines)

local out = assert(io.open(outPath, "w"))
for i = 1, #lines, stride do
	local line = lines[i]
	local rec = { line = line, options = {} }
	for _, modType in ipairs({ "explicit", "implicit", "enchant", "scourge" }) do
		local id, value = tradeHelpers.findTradeIdOption(line, modType)
		if id then
			table.insert(rec.options, { type = modType, id = id, value = value })
		end
	end
	local hashes, value, negate = tradeHelpers.findTradeHash(line)
	rec.hashes = hashes
	rec.value = value
	if negate == nil then
		rec.negate = "nil"
	else
		rec.negate = negate
	end
	out:write(json.encode(rec), "\n")
end
out:close()
