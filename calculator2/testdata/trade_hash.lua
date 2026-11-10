-- usage (from PoB-PoE2's src/): luajit trade_hash.lua <out.jsonl> <stride>
-- Records TradeHelpers' option and stat description matches for every <stride>-th distinct line of the item mod
-- tables, ranges at their midpoint, sorted: the option entry per mod type, the trade hashes, the value and whether the
-- trade site wants it negated ("nil" when no form said).
local outPath, stride = arg[1], tonumber(arg[2] or "1")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local tradeHelpers = LoadModule("Classes/TradeHelpers")

local seen, lines = {}, {}
for _, key in ipairs({ "Item", "Flask", "Charm", "Jewel", "Corruption", "Exclusive", "Desecrated" }) do
	for _, mod in pairs(data.itemMods[key]) do
		for _, line in ipairs(mod) do
			local resolved = itemLib.applyRange(line, 0.5)
			if not seen[resolved] then
				seen[resolved] = true
				table.insert(lines, resolved)
			end
		end
	end
end
table.sort(lines)

local out = assert(io.open(outPath, "w"))
for i = 1, #lines, stride do
	local line = lines[i]
	local rec = { line = line, options = {} }
	for _, modType in ipairs({ "explicit", "implicit", "enchant" }) do
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
