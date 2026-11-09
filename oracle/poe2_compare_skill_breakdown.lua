-- usage (from PoB-PoE2's src/): luajit poe2_compare_skill_breakdown.lua <build.xml> <out.json>
-- Records the rows the Compare page's Calcs header draws for a build's SkillBuffs and SkillDebuffs panels
-- (CompareCalcsHelpers.DrawSkillBreakdownPanel over its CALCS calculation): each row's Value, Name, Source and
-- SourceName cells, read from the strings it draws.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_compare_skill_breakdown.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local helpers = LoadModule("Classes/CompareCalcsHelpers")
local keys = { "Value", "Name", "Source", "SourceName" }
local out = {}
local drawString = DrawString
for _, name in ipairs({ "SkillBuffs", "SkillDebuffs" }) do
	local headers, cells = {}, {}
	DrawString = function(x, y, align, size, font, text)
		if size == 16 then
			headers[#headers + 1] = x
		elseif size == 12 then
			cells[y] = cells[y] or {}
			for i, hx in ipairs(headers) do
				if hx == x then
					cells[y][keys[i]] = text:gsub("^%^7", "", 1)
				end
			end
		end
	end
	helpers.DrawSkillBreakdownPanel(build, name, name, 0, 0, 10, 10, { x = 0, y = 0, width = 100000, height = 100000 })
	local ys = {}
	for y in pairs(cells) do
		ys[#ys + 1] = y
	end
	table.sort(ys)
	local rows = {}
	for _, y in ipairs(ys) do
		local row = cells[y]
		for _, k in ipairs(keys) do
			row[k] = row[k] or ""
		end
		rows[#rows + 1] = row
	end
	out[name] = rows
end
DrawString = drawString

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
