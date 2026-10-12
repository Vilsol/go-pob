local buildsDir = arg[1]
assert(buildsDir, "usage: compare_skill_breakdown.lua <buildsDir> [<k>/<n>]")
local shard, shards = (arg[2] or "0/1"):match("^(%d+)/(%d+)$")
shard, shards = tonumber(shard), tonumber(shards)
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local lfs = require("lfs")
local helpers = LoadModule("Classes/CompareCalcsHelpers")

local builds = { }
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)

local drawn
local realDrawString = DrawString
local function panel(key)
	drawn = { }
	DrawString = function(x, y, align, size, font, str)
		drawn[#drawn + 1] = { x = x, y = y, str = str }
	end
	helpers.DrawSkillBreakdownPanel(build, key, key, 0, 0, 10, 10, { x = 0, y = 0, width = 100000, height = 1000000 })
	DrawString = realDrawString
	if #drawn == 0 then
		return { }
	end
	local cols, rows, byY = { }, { }, { }
	for i = 1, 4 do
		cols[drawn[i].x] = i
	end
	for i = 5, #drawn do
		local d = drawn[i]
		local row = byY[d.y]
		if not row then
			row = { "", "", "", "" }
			byY[d.y] = row
			rows[#rows + 1] = { y = d.y, cells = row }
		end
		row[cols[d.x]] = d.str:gsub("^%^7", "", 1)
	end
	table.sort(rows, function(a, b) return a.y < b.y end)
	local out = { }
	for i, r in ipairs(rows) do
		out[i] = table.concat(r.cells, "\t")
	end
	return out
end

for i, file in ipairs(builds) do
	if i % 40 == 1 and ((i - 1) / 40) % shards == shard then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			print(json.encode({ build = file, SkillBuffs = panel("SkillBuffs"), SkillDebuffs = panel("SkillDebuffs") }, { keyorder = { "build", "SkillBuffs", "SkillDebuffs" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
