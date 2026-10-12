local buildsDir = arg[1]
assert(buildsDir, "usage: calcs_skill_buffs.lua <buildsDir>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local lfs = require("lfs")

local builds = { }
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)

local function rows(breakdown)
	local out = { }
	for _, row in ipairs(breakdown and breakdown.modList or { }) do
		local value = type(row.value) == "number" and string.format("%.4f", row.value) or tostring(row.value)
		out[#out + 1] = row.mod.name .. "|" .. row.mod.type .. "|" .. value .. "|" .. row.mod.source
	end
	table.sort(out)
	return out
end

for i, file in ipairs(builds) do
	if i % 40 == 1 then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			local bd = build.calcsTab.calcsEnv.player.breakdown
			print(json.encode({ build = file, SkillBuffs = rows(bd.SkillBuffs), SkillDebuffs = rows(bd.SkillDebuffs) }, { keyorder = { "build", "SkillBuffs", "SkillDebuffs" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
