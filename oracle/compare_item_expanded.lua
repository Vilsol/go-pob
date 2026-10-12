local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local tradeHelpers = LoadModule("Classes/TradeHelpers")
LoadModule("Classes/CompareTab")
local CompareTab = common.classes["CompareTab"]
assert(CompareTab and CompareTab.DrawItemExpanded, "REFUSING: CompareTab.DrawItemExpanded not loaded")

local drawn
local realDrawString = DrawString
DrawString = function(x, y, align, size, font, str)
	table.insert(drawn, str)
end
local card = function(item, other)
	drawn = {}
	CompareTab.DrawItemExpanded({}, item, 0, 0, 100, tradeHelpers.buildModMap(other))
	return drawn
end

local pairsIn = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, p in ipairs(pairsIn) do
	local a = p[1] ~= "" and new("Item", p[1]) or nil
	local b = p[2] ~= "" and new("Item", p[2]) or nil
	out[k] = { card(a, b), card(b, a) }
end
DrawString = realDrawString
local f = io.open(outPath, "w")
f:write(json.encode(out, { indent = true }))
f:close()
