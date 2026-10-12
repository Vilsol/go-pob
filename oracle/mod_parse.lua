
local selfPath = arg[0]
local casesPath, outPath, reportPath = arg[1], arg[2], arg[3]
assert(casesPath and outPath, "usage: mod_parse.lua <cases.json> <out.json> [report.json]")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"))
local cases = assert(json.decode(f:read("*a")))
f:close()

local realParseMod = modLib.parseMod
local modCache = modLib.parseModCache

local shipped, shippedCount = {}, 0
for k in pairs(modCache) do
	shipped[k] = true
	shippedCount = shippedCount + 1
end

local function verdict(modList, extra)
	local names = {}
	if modList then
		for i, m in ipairs(modList) do
			names[i] = m.name or (m[1] and m[1].name) or "?"
		end
	end
	return {
		parsed = modList ~= nil and modList ~= false,
		extra = extra,
		count = modList and #modList or 0,
		names = names,
	}
end

local function sameVerdict(a, b)
	return a.parsed == b.parsed and a.extra == b.extra and a.count == b.count
end

local calls = {}
local order = {}
local lineOrder = {}
local lineSeen = {}

modLib.parseMod = function(line, isComb)
	local modList, extra = realParseMod(line, isComb)
	if type(line) == "string" then
		local key = tostring(isComb) .. "\0" .. line
		local v = verdict(modList, extra)
		local rec = calls[key]
		if rec == nil then
			calls[key] = { line = line, isComb = isComb, observed = v, hits = 1, unstable = false }
			table.insert(order, key)
		else
			rec.hits = rec.hits + 1
			if not sameVerdict(rec.observed, v) then
				rec.unstable = true
			end
		end
		if not lineSeen[line] then
			lineSeen[line] = true
			table.insert(lineOrder, line)
		end
	end
	return modList, extra
end

for _, c in ipairs(cases) do
	pcall(function()
		new("Item", c.raw)
	end)
end

modLib.parseMod = realParseMod

local live = {}
iscombComparisons = 0
for _, key in ipairs(order) do
	local rec = calls[key]
	modCache[rec.line] = nil
	local modList, extra = realParseMod(rec.line, rec.isComb)
	rec.live = verdict(modList, extra)
	if live[rec.line] == nil then
		live[rec.line] = rec.live
	else
		iscombComparisons = iscombComparisons + 1
		if not sameVerdict(live[rec.line], rec.live) then
			rec.iscombDiffers = true
		end
	end
end

local results = {}
for i, line in ipairs(lineOrder) do
	local v = live[line]
	results[i] = { line = line, parsed = v.parsed, extra = v.extra, count = v.count, names = v.names }
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()

local stats = { calls = 0, distinctKeys = #order, distinctLines = #lineOrder,
	shippedCacheKeys = shippedCount, servedFromShippedCache = 0, cacheDiffers = 0,
	unstableWithinPass = 0, iscombDiffers = 0, iscombComparisons = iscombComparisons }
local divergences = {}
for _, key in ipairs(order) do
	local rec = calls[key]
	stats.calls = stats.calls + rec.hits
	if rec.unstable then stats.unstableWithinPass = stats.unstableWithinPass + 1 end
	if rec.iscombDiffers then stats.iscombDiffers = stats.iscombDiffers + 1 end
	if shipped[rec.line] then
		stats.servedFromShippedCache = stats.servedFromShippedCache + 1
		if not sameVerdict(rec.observed, rec.live) then
			stats.cacheDiffers = stats.cacheDiffers + 1
			table.insert(divergences, { line = rec.line, isComb = rec.isComb,
				shippedCache = rec.observed, liveGrammar = rec.live })
		end
	end
end

if reportPath then
	local rf = assert(io.open(reportPath, "w"))
	rf:write(json.encode(stable({ stats = stats, cacheDivergences = divergences }), { indent = true }))
	rf:close()
end

io.stderr:write(string.format(
	"oracle: %d parseMod calls, %d distinct (line,isComb) keys, %d distinct lines -> %s\n",
	stats.calls, stats.distinctKeys, stats.distinctLines, outPath))
io.stderr:write(string.format(
	"oracle: shipped ModCache %d keys; %d of our lines were served from it, %d of those disagree with the live grammar\n",
	stats.shippedCacheKeys, stats.servedFromShippedCache, stats.cacheDiffers))
io.stderr:write(string.format(
	"oracle: %d keys unstable within pass 1; isComb changed the live verdict on %d of %d lines reached both ways\n",
	stats.unstableWithinPass, stats.iscombDiffers, stats.iscombComparisons))
