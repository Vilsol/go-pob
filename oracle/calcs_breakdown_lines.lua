local buildsDir = arg[1]
assert(buildsDir, "usage: calcs_breakdown_lines.lua <buildsDir>")
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

local function lines(out, prefix, bd)
	if not bd then
		return
	end
	for key, b in pairs(bd) do
		if type(key) == "string" and type(b) == "table" then
			if key == "MainHand" or key == "OffHand" then
				if prefix == "" then
					lines(out, key .. ".", b)
				end
			else
				local l = { }
				for _, s in ipairs(b) do
					if type(s) == "string" then
						l[#l + 1] = s
					end
				end
				if #l > 0 then
					out[prefix .. key] = l
				end
			end
		end
	end
end

for i, file in ipairs(builds) do
	if i % 40 == 1 then
		local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local ok, err = pcall(function()
			loadBuildFromXML(xmlText, file)
			runCallback("OnFrame")
			local env = build.calcsTab.calcsEnv
			local player, minion = { }, { }
			lines(player, "", env.player.breakdown)
			if env.minion then
				lines(minion, "", env.minion.breakdown)
			end
			print(json.encode({ build = file, player = player, minion = minion }, { keyorder = { "build", "player", "minion" } }))
		end)
		if not ok then
			io.stderr:write(file .. ": " .. tostring(err) .. "\n")
		end
	end
end
