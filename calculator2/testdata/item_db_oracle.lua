-- Prints PoB-PoE2's unique database as Main.lua loads it, one JSON row per item by name: its raw text (after the
-- runeforging base variants), type, primary slot, league, source and level requirement. Then the database list's
-- League and Type dropdowns, the character level and attributes the requirement filter reads, and the names ItemDBControl lists for a few filter settings.
-- usage: item_db_oracle.lua <out.jsonl>
local outPath = arg[1]
assert(outPath, "usage: item_db_oracle.lua <out.jsonl>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeList(list)
	local parts = {}
	for i, s in ipairs(list) do
		parts[i] = encodeString(s)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

newBuild()
local build = __mainObject__.main.modes.BUILD
local out = assert(io.open(outPath, "w"))

local names = {}
for name in pairs(main.uniqueDB.list) do
	names[#names + 1] = name
end
table.sort(names)
for _, name in ipairs(names) do
	local item = main.uniqueDB.list[name]
	out:write('{"name":', encodeString(name), ',"raw":', encodeString(item.raw), ',"type":', encodeString(item.type),
		',"primarySlot":', encodeString(item:GetPrimarySlot() or ""), ',"league":', encodeString(item.league or ""),
		',"source":', encodeString(item.source or ""), ',"level":', item.requirements.level or 0, "}\n")
end

local db = build.itemsTab.controls.uniqueDB
db:LoadLeaguesAndTypes()
out:write('{"leagues":', encodeList(db.leagueList), ',"types":', encodeList(db.typeList), "}\n")
local mainOutput = build.calcsTab.mainOutput
out:write('{"context":[', table.concat({ build.characterLevel, mainOutput.Str, mainOutput.Dex, mainOutput.Int }, ","), "]}\n")

-- slot, type, league, requirement, obtainable, search, search mode
local filters = {
	{ 1, 1, 1, 1, 1, "", 1 },
	{ 1, 1, 1, 1, 2, "", 1 },
	{ 1, 1, 1, 1, 3, "", 1 },
	{ 1, 1, 2, 1, 2, "", 1 },
	{ 1, 1, 3, 1, 2, "", 1 },
	{ 2, 1, 1, 1, 2, "", 1 },
	{ 3, 1, 1, 1, 2, "", 1 },
	{ 9, 1, 1, 1, 2, "", 1 },
	{ 1, 2, 1, 1, 2, "", 1 },
	{ 1, 3, 1, 1, 2, "", 1 },
	{ 1, 4, 1, 1, 2, "", 1 },
	{ 1, 5, 1, 1, 2, "", 1 },
	{ 1, 8, 1, 1, 2, "", 1 },
	{ 1, 1, 1, 2, 2, "", 1 },
	{ 1, 1, 1, 3, 2, "", 1 },
	{ 1, 1, 1, 1, 6, "", 1 },
	{ 1, 1, 1, 1, 2, "life", 1 },
	{ 1, 1, 1, 1, 2, "life", 2 },
	{ 1, 1, 1, 1, 2, "life", 3 },
	{ 1, 1, 1, 1, 2, "(fire|cold) res", 1 },
	{ 1, 1, 1, 1, 2, "the", 2 },
	{ 1, 1, 1, 1, 2, "+1 to level", 3 },
	{ 1, 1, 1, 1, 2, "critical hit chance", 3 },
	{ 1, 1, 1, 1, 2, "lifeonhit", 3 },
}
for _, f in ipairs(filters) do
	db.controls.slot.selIndex = f[1]
	db.controls.type.selIndex = f[2]
	db.controls.league.selIndex = f[3]
	db.controls.requirement.selIndex = f[4]
	db.controls.obtainable.selIndex = f[5]
	db.controls.search:SetText(f[6])
	db.controls.searchMode.selIndex = f[7]
	db:ListBuilder()
	local listed = {}
	for i, item in ipairs(db.list) do
		listed[i] = item.name
	end
	out:write('{"filter":[', table.concat({ f[1], f[2], f[3], f[4], f[5] }, ","), '],"search":', encodeString(f[6]),
		',"mode":', f[7], ',"listed":', encodeList(listed), "}\n")
end
out:close()
