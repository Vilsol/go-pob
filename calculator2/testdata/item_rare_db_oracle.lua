-- Prints PoB-PoE2's rare database as Main.lua loads it, one JSON row per item by name: its text as BuildRaw writes it
-- after Craft, type, primary slot and level requirement. Then the database list's Type dropdown.
-- usage: item_rare_db_oracle.lua <out.jsonl>
local outPath = arg[1]
assert(outPath, "usage: item_rare_db_oracle.lua <out.jsonl>")

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
for name in pairs(main.rareDB.list) do
	names[#names + 1] = name
end
table.sort(names)
for _, name in ipairs(names) do
	local item = main.rareDB.list[name]
	out:write('{"name":', encodeString(name), ',"raw":', encodeString(item:BuildRaw()), ',"type":', encodeString(item.type),
		',"primarySlot":', encodeString(item:GetPrimarySlot() or ""), ',"level":', item.requirements.level or 0, "}\n")
end

local db = build.itemsTab.controls.rareDB
db:LoadLeaguesAndTypes()
out:write('{"types":', encodeList(db.typeList), "}\n")
out:close()
