
local buildsDir, outDir, listFile = arg[1], arg[2], arg[3]
assert(buildsDir and outDir, "usage: dump_calcs_damage.lua <buildsDir> <outDir> [listFile]")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local function numbers(t)
	local out = {}
	if type(t) ~= "table" then
		return out
	end
	for k, v in pairs(t) do
		if type(k) == "string" and type(v) == "number" then
			out[k] = v
		end
	end
	return out
end

local function damageProducts(bd)
	if type(bd) ~= "table" then
		return nil
	end
	local out = {}
	for dt, entry in pairs(bd) do
		if type(dt) == "string" and type(entry) == "table" and type(entry.damageTypes) == "table" then
			for _, row in ipairs(entry.damageTypes) do
				if type(row) == "table" and type(row.source) == "string" then
					local function factor(v)
						if v == false or v == nil then
							return 1
						end
						return tonumber(tostring(v):match("^x%s*(.+)$")) or 1
					end
					if row.inc == nil and row.more == nil then
					else
						out[dt .. "/" .. row.source] = { inc = factor(row.inc), more = factor(row.more) }
					end
				end
			end
		end
	end
	return next(out) and out or nil
end

local builds = {}
if listFile then
	for line in assert(io.lines(listFile)) do
		local name = line:match("^%s*(.-)%s*$")
		if name ~= "" and not name:match("^#") then
			builds[#builds + 1] = name
		end
	end
else
	for file in lfs.dir(buildsDir) do
		if file:match("%.xml$") then
			builds[#builds + 1] = file
		end
	end
end
table.sort(builds)

lfs.mkdir(outDir)

for _, file in ipairs(builds) do
	local path = buildsDir .. "/" .. file
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		local mainEnv = build.calcsTab.mainEnv
		build.calcsTab.input.skill_number = build.mainSocketGroup
		local mainGroup = build.skillsTab.socketGroupList[build.mainSocketGroup]
		if mainGroup then
			mainGroup.mainActiveSkillCalcs = mainGroup.mainActiveSkill
		end
		build.calcsTab:BuildOutput()

		local calcsEnv = build.calcsTab.calcsEnv
		local output = calcsEnv.player.output

		local function skillName(e)
			return e and e.player.mainSkill and e.player.mainSkill.activeEffect
				and e.player.mainSkill.activeEffect.grantedEffect
				and e.player.mainSkill.activeEffect.grantedEffect.name
		end
		entry.gradedSkill = skillName(mainEnv)
		entry.skillMismatch = (skillName(calcsEnv) ~= entry.gradedSkill)

		entry.output = numbers(output)
		entry.outputMainHand = output.MainHand and numbers(output.MainHand) or nil
		entry.outputOffHand = output.OffHand and numbers(output.OffHand) or nil
		entry.mainSkill = calcsEnv.player.mainSkill and calcsEnv.player.mainSkill.activeEffect
			and calcsEnv.player.mainSkill.activeEffect.grantedEffect
			and calcsEnv.player.mainSkill.activeEffect.grantedEffect.name

		local bd = calcsEnv.player.breakdown
		entry.productsMainHand = bd and damageProducts(bd.MainHand) or nil
		entry.productsOffHand = bd and damageProducts(bd.OffHand) or nil
	end)
	if not ok then
		entry.error = tostring(err)
	end

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry))
	out:close()

	io.stderr:write(string.format("calcs-dump: %s%s\n", file,
		entry.error and ("  ERROR " .. entry.error) or ""))
end

io.stderr:write("calcs-dump: wrote " .. #builds .. " builds to " .. outDir .. "\n")
