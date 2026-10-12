
local buildsDir, outDir, listFile = arg[1], arg[2], arg[3]
assert(buildsDir and outDir, "usage: dump_conversion.lua <buildsDir> <outDir> [listFile]")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local dmgTypeList = { "Physical", "Lightning", "Cold", "Fire", "Chaos" }

local function conversionOf(t)
	if type(t) ~= "table" then
		return nil
	end
	local out = {}
	for _, dt in ipairs(dmgTypeList) do
		local e = t[dt]
		if type(e) == "table" then
			local row = { mult = e.mult }
			for _, other in ipairs(dmgTypeList) do
				local conv = e.conversion and e.conversion[other]
				local gain = e.gain and e.gain[other]
				local combined = e[other]
				if (conv and conv ~= 0) or (gain and gain ~= 0) or (type(combined) == "number" and combined ~= 0) then
					row[other] = { conversion = conv, gain = gain, combined = combined }
				end
			end
			out[dt] = row
		end
	end
	return out
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
		local skill = mainEnv.player.mainSkill

		if skill then
			entry.mainSkill = skill.activeEffect and skill.activeEffect.grantedEffect
				and skill.activeEffect.grantedEffect.name
			entry.skillCfg = conversionOf(skill.conversionTable)
			entry.weapon1Cfg = skill.weapon1Cfg and conversionOf(skill.weapon1Cfg.conversionTable) or nil
			entry.weapon2Cfg = skill.weapon2Cfg and conversionOf(skill.weapon2Cfg.conversionTable) or nil
			entry.weapon1HasOwnTable = (skill.weapon1Cfg ~= nil) and (skill.weapon1Cfg.conversionTable ~= nil) or false
			entry.weapon2HasOwnTable = (skill.weapon2Cfg ~= nil) and (skill.weapon2Cfg.conversionTable ~= nil) or false
			local ce = build.calcsTab.calcsEnv
			entry.calcsTabSkill = ce and ce.player.mainSkill and ce.player.mainSkill.activeEffect
				and ce.player.mainSkill.activeEffect.grantedEffect
				and ce.player.mainSkill.activeEffect.grantedEffect.name
			entry.skillMismatch = (entry.calcsTabSkill ~= entry.mainSkill)
		end
	end)
	if not ok then
		entry.error = tostring(err)
	end

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry, { indent = true }))
	out:close()

	io.stderr:write(string.format("conversion-dump: %s%s\n", file,
		entry.error and ("  ERROR " .. entry.error) or ""))
end

io.stderr:write("conversion-dump: wrote " .. #builds .. " builds to " .. outDir .. "\n")
