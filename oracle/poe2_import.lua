-- usage (cwd = <PoB-PoE2>/src): luajit poe2_import.lua <character.json> <skills-on.xml> <skills-off.xml> <row.tsv>
-- Imports one character JSON through ImportTab, picks the socket group with the highest CombinedDPS as the main
-- skill, and saves the build twice: as imported, and with every socket group disabled. Appends one TSV row.
local jsonPath, onPath, offPath, rowPath = arg[1], arg[2], arg[3], arg[4]
assert(jsonPath and onPath and offPath and rowPath, "usage: poe2_import.lua <character.json> <skills-on.xml> <skills-off.xml> <row.tsv>")

local conLog = {}
dofile("HeadlessWrapper.lua")
function ConPrintf(fmt, ...) conLog[#conLog + 1] = string.format(fmt, ...) end
local dkjson = require "dkjson"

local function dps()
	local o = build.calcsTab.mainOutput or {}
	return o.CombinedDPS or o.TotalDPS or 0
end

local function skillName()
	local e = build.calcsTab.mainEnv
	local m = e and e.player and e.player.mainSkill
	return m and m.activeEffect and m.activeEffect.grantedEffect and m.activeEffect.grantedEffect.name or ""
end

local function write(path, text)
	local f = assert(io.open(path, "w"))
	f:write(text)
	f:close()
end

local t0 = os.clock()
local r = { stage = "read" }
local ok, err = pcall(function()
	local fh = assert(io.open(jsonPath, "rb"))
	local body = fh:read("*a")
	fh:close()
	local doc, _, perr = dkjson.decode(body)
	assert(doc and doc.character, "json: " .. tostring(perr or "no character"))
	local ch = doc.character
	r.timeless = 0
	for _, j in ipairs(ch.jewels or {}) do
		if tostring(j.baseType or j.typeLine or ""):match("Timeless") then r.timeless = r.timeless + 1 end
	end

	r.stage = "import"
	newBuild()
	build.importTab:ImportItemsAndSkills(ch)
	build.importTab:ImportPassiveTreeAndJewels(ch)
	runCallback("OnFrame")

	r.stage = "calc"
	build.calcsTab:BuildOutput()
	runCallback("OnFrame")
	r.groups = #build.skillsTab.socketGroupList
	r.guess, r.guessSkill, r.guessDPS = build.mainSocketGroup, skillName(), dps()
	r.pick, r.pickSkill, r.pickDPS = r.guess, r.guessSkill, r.guessDPS

	r.stage = "sweep"
	r.sweepErr = 0
	for i, g in ipairs(build.skillsTab.socketGroupList) do
		if g.enabled ~= false and i ~= r.guess then
			build.mainSocketGroup = i
			if pcall(build.calcsTab.BuildOutput, build.calcsTab) then
				local d = dps()
				if d > r.pickDPS then r.pick, r.pickSkill, r.pickDPS = i, skillName(), d end
			else
				r.sweepErr = r.sweepErr + 1
			end
		end
	end
	build.mainSocketGroup = r.pick
	build.calcsTab:BuildOutput()
	runCallback("OnFrame")

	r.stage = "save"
	write(onPath, build:SaveDB("skills-on"))
	for _, g in ipairs(build.skillsTab.socketGroupList) do g.enabled = false end
	write(offPath, build:SaveDB("skills-off"))
	r.stage = "done"
end)

local function s(v) return v == nil and "" or (tostring(v):gsub("[\t\r\n]", " ")) end
local name = jsonPath:match("([^/\\]+)%.json$") or jsonPath
local row = { name, ok and 1 or 0, ok and "" or r.stage, ok and "" or s(err):sub(1, 300), s(r.groups), s(r.guess), s(r.guessSkill),
	s(r.guessDPS), s(r.pick), s(r.pickSkill), s(r.pickDPS), s(r.sweepErr), s(r.timeless), string.format("%.2f", os.clock() - t0) }
local out = assert(io.open(rowPath, "a"))
out:write(table.concat(row, "\t"), "\n")
out:close()
for _, l in ipairs(conLog) do io.stderr:write(name, ": ", l, "\n") end
os.exit(ok and 0 or 1)
