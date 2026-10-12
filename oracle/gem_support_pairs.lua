local buildsDir = arg[1]
assert(buildsDir, "usage: gem_support_pairs.lua <buildsDir>")
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
	if i % 100 == 1 then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			local supporting = build.skillsTab.controls.imbuedSupport.CheckSupporting
			local groups = { }
			for gi, group in ipairs(build.skillsTab.socketGroupList) do
				local pairs_ = { }
				for a, gemA in ipairs(group.gemList) do
					for b, gemB in ipairs(group.gemList) do
						if a ~= b and gemA.enabled and gemB.enabled and gemA.gemData and gemB.gemData and supporting(nil, gemA, gemB) then
							pairs_[#pairs_ + 1] = { a - 1, b - 1 }
						end
					end
				end
				groups[gi] = pairs_
			end
			print(json.encode({ build = file, groups = groups }, { keyorder = { "build", "groups" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
