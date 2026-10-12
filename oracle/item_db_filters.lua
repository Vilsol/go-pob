local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local cases = json.decode(assert(io.open(inPath)):read("*a"))
local itemsTab = build.itemsTab
local out = { cases = cases, results = {} }
for _, dbName in ipairs({ "uniqueDB", "rareDB" }) do
	local db = itemsTab.controls[dbName]
	db:LoadLeaguesAndTypes()
	out[dbName] = { leagues = db.leagueList, types = db.typeList }
end
for i, c in ipairs(cases) do
	local db = itemsTab.controls[c.db == "RARE" and "rareDB" or "uniqueDB"]
	db.controls.slot.selIndex = c.slot
	db.controls.type.selIndex = c.type
	if db.controls.league then
		db.controls.league.selIndex = c.league
		db.controls.requirement.selIndex = c.requirement
		db.controls.obtainable.selIndex = c.obtainable
	end
	db.controls.search.buf = c.search
	db.controls.searchMode.selIndex = c.searchMode
	build.characterLevel = c.level
	build.calcsTab.mainOutput.Str = c.str
	build.calcsTab.mainOutput.Dex = c.dex
	build.calcsTab.mainOutput.Int = c.int
	db:ListBuilder()
	local names = {}
	for k, item in ipairs(db.list) do
		names[k] = item.name
	end
	out.results[i] = names
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out, { indent = true }))
f:close()
