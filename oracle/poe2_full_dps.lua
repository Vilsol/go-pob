-- usage (from PoB-PoE2's src/): luajit poe2_full_dps.lua <out.jsonl> <build.xml>...
-- Includes every socket group of each build in Full DPS and records the Full DPS PoB-PoE2's buildOutput calculates
-- (FullDPS, FullDotDPS and every SkillDPS row), the sidebar rows AddDisplayStatList shows from Full DPS on and the
-- FullDPSSkill rows Save writes, one JSON line per build.
local outPath, paths = arg[1], { select(2, unpack(arg)) }
assert(outPath and paths[1], "usage: poe2_full_dps.lua <out.jsonl> <build.xml>...")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function plain(s)
	return (s:gsub("%^x%x%x%x%x%x%x", ""):gsub("%^%d", ""))
end

local out = assert(io.open(outPath, "w"))
for _, path in ipairs(paths) do
	local f = assert(io.open(path))
	local xml = f:read("*a")
	f:close()
	xml = xml:gsub(' includeInFullDPS="[^"]*"', ""):gsub("<Skill ", '<Skill includeInFullDPS="true" ')
	local rec = { build = path:match("([^/\\]+)$") }
	local ok, err = pcall(function()
		-- a build loaded after another one otherwise reads the earlier build's GlobalCache entries
		wipeGlobalCache()
		loadBuildFromXML(xml)
		runCallback("OnFrame")
		local output = build.calcsTab.mainOutput
		rec.FullDPS = output.FullDPS
		rec.FullDotDPS = output.FullDotDPS
		rec.skills = {}
		for _, s in ipairs(output.SkillDPS or {}) do
			rec.skills[#rec.skills + 1] = { name = s.name, dps = s.dps, count = s.count, trigger = s.trigger, skillPart = s.skillPart, source = s.source }
		end
		local rows = {}
		for _, row in ipairs(build.controls.statBox.list) do
			if row[1] and (#rows > 0 or plain(row[1]) == "Full DPS:") then
				rows[#rows + 1] = { plain(row[1]), plain(row[2] or "") }
			end
		end
		rec.statBox = rows
		local saved = build:SaveDB("fixture")
		local fullRows = {}
		for attrs in saved:gmatch("<FullDPSSkill ([^>]*)/>") do
			local r = {}
			for k, v in attrs:gmatch('(%w+)="([^"]*)"') do
				r[k] = v
			end
			fullRows[#fullRows + 1] = r
		end
		rec.saved = fullRows
	end)
	if not ok then
		rec.error = tostring(err)
	end
	out:write(json.encode(rec), "\n")
	out:flush()
	-- each loaded build leaves a large garbage heap the incremental collector does not keep up with
	collectgarbage("collect")
end
out:close()
