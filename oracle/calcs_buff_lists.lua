local buildsDir = arg[1]
assert(buildsDir, "usage: calcs_buff_lists.lua <buildsDir>")
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
for i, file in ipairs(builds) do
	if i % 20 == 1 then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			local out = build.calcsTab.calcsOutput
			local entry = { build = file, BuffList = out.BuffList, CombatList = out.CombatList, CurseList = out.CurseList }
			if out.Minion then
				entry.Minion = { BuffList = out.Minion.BuffList, CombatList = out.Minion.CombatList, CurseList = out.Minion.CurseList }
			end
			print(json.encode(entry, { keyorder = { "build", "BuffList", "CombatList", "CurseList", "Minion" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
