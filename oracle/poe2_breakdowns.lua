-- usage (from PoB-PoE2's src/): luajit poe2_breakdowns.lua <build.xml> <out.json>
-- Records, for every breakdown the main calculation's player holds, the sections CalcBreakdownControl builds for
-- it: TEXT lines, TABLE label, footer, columns and rows (each drawn cell as tostring gives it) and RADIUS. A mod list
-- section is recorded as MODS with its row count only. A breakdown with slots is recorded again under
-- "<name>#gearOnly" as a gearOnly cell shows it; an attack pass's own breakdown is "MainHand.<name>" or
-- "OffHand.<name>".
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_breakdowns.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local calcsTab = build.calcsTab
local control = calcsTab.controls.breakdown
control.envName = "mainEnv"
local actor = calcsTab.mainEnv.player

local function record(name, gearOnly)
	control:SetBreakdownData({ { breakdown = name, gearOnly = gearOnly } })
	local sections = {}
	for _, s in ipairs(control.shown and control.sectionList or {}) do
		if s.type == "TEXT" then
			local lines = {}
			for i, l in ipairs(s.lines) do
				lines[i] = tostring(l)
			end
			sections[#sections + 1] = { type = "TEXT", lines = lines }
		elseif s.type == "RADIUS" then
			sections[#sections + 1] = { type = "RADIUS", radius = s.radius }
		elseif s.type == "TABLE" and s.colList[1] and s.colList[1].key == "displayValue" then
			sections[#sections + 1] = { type = "MODS", rows = #s.rowList }
		elseif s.type == "TABLE" then
			local cols, rows = {}, {}
			for i, c in ipairs(s.colList) do
				cols[i] = { label = c.label, key = c.key }
			end
			for i, r in ipairs(s.rowList) do
				local row = {}
				for _, c in ipairs(s.colList) do
					if r[c.key] ~= nil and r[c.key] ~= false then
						row[c.key] = tostring(r[c.key])
					end
				end
				rows[i] = row
			end
			sections[#sections + 1] = { type = "TABLE", label = s.label, footer = s.footer, columns = cols, rows = rows }
		end
	end
	return #sections > 0 and sections or nil
end

local out = {}
local function recordAll(prefix, breakdown)
	for name, bd in pairs(breakdown) do
		if type(name) == "string" and type(bd) == "table" and name ~= "MainHand" and name ~= "OffHand" then
			out[prefix .. name] = record(prefix .. name)
			if bd.slots and #bd.slots > 0 then
				out[prefix .. name .. "#gearOnly"] = record(prefix .. name, true)
			end
		end
	end
end
recordAll("", actor.breakdown)
for _, hand in ipairs({ "MainHand", "OffHand" }) do
	if actor.breakdown[hand] then
		recordAll(hand .. ".", actor.breakdown[hand])
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
