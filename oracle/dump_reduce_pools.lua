
local buildsDir, outDir, listFile = arg[1], arg[2], arg[3]
assert(buildsDir and outDir, "usage: dump_reduce_pools.lua <buildsDir> <outDir> [listFile]")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local calcs = LoadModule("Modules/Calcs")

local dmgTypeList = { "Physical", "Lightning", "Cold", "Fire", "Chaos" }

local function poolsOf(p)
	if type(p) ~= "table" then
		return nil
	end
	local out = {
		Ward = p.Ward,
		WardActiveChance = p.WardActiveChance,
		EnergyShield = p.EnergyShield,
		Mana = p.Mana,
		Life = p.Life,
		LifeLossLostOverTime = p.LifeLossLostOverTime,
		LifeBelowHalfLossLostOverTime = p.LifeBelowHalfLossLostOverTime,
		OverkillDamage = p.OverkillDamage,
		hitPoolRemaining = p.hitPoolRemaining,
	}
	if type(p.Aegis) == "table" then
		out.Aegis = { shared = p.Aegis.shared, sharedElemental = p.Aegis.sharedElemental }
		for _, dt in ipairs(dmgTypeList) do
			out.Aegis[dt] = p.Aegis[dt]
		end
	end
	if type(p.Guard) == "table" then
		out.Guard = { shared = p.Guard.shared }
		for _, dt in ipairs(dmgTypeList) do
			out.Guard[dt] = p.Guard[dt]
		end
	end
	if type(p.damageTakenThatCanBeRecouped) == "table" then
		out.damageTakenThatCanBeRecouped = {}
		for _, dt in ipairs(dmgTypeList) do
			out.damageTakenThatCanBeRecouped[dt] = p.damageTakenThatCanBeRecouped[dt]
		end
	end
	if type(p.AlliesTakenBeforeYou) == "table" then
		out.AlliesTakenBeforeYou = {}
		for key, v in pairs(p.AlliesTakenBeforeYou) do
			if type(v) == "table" then
				out.AlliesTakenBeforeYou[key] = { remaining = v.remaining, percent = v.percent, damageType = v.damageType }
			end
		end
	end
	if type(p.resourcesLostToTypeDamage) == "table" then
		out.resourcesLostToTypeDamage = {}
		for _, dt in ipairs(dmgTypeList) do
			local r = p.resourcesLostToTypeDamage[dt]
			if type(r) == "table" and next(r) ~= nil then
				out.resourcesLostToTypeDamage[dt] = r
			end
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
		local actor = mainEnv.player
		local output = actor.output

		entry.inputs = {
			Life = output.Life,
			LifeRecoverable = output.LifeRecoverable,
			ManaUnreserved = output.ManaUnreserved,
			EnergyShieldRecoveryCap = output.EnergyShieldRecoveryCap,
			Ward = output.Ward,
			preventedLifeLoss = output.preventedLifeLoss,
			preventedLifeLossBelowHalf = output.preventedLifeLossBelowHalf,
			preventedLifeLossTotal = output.preventedLifeLossTotal,
			sharedMindOverMatter = output.sharedMindOverMatter,
			sharedAegis = output.sharedAegis,
			sharedElementalAegis = output.sharedElementalAegis,
			sharedGuardAbsorb = output.sharedGuardAbsorb,
			sharedGuardAbsorbRate = output.sharedGuardAbsorbRate,
			LifeLossBelowHalfPrevented = actor.modDB:Sum("BASE", nil, "LifeLossBelowHalfPrevented"),
			EnergyShieldProtectsMana = actor.modDB:Flag(nil, "EnergyShieldProtectsMana") or false,
		}
		for _, dt in ipairs(dmgTypeList) do
			entry.inputs[dt .. "TakenHit"] = output[dt .. "TakenHit"]
			entry.inputs[dt .. "MindOverMatter"] = output[dt .. "MindOverMatter"]
			entry.inputs[dt .. "EnergyShieldBypass"] = output[dt .. "EnergyShieldBypass"]
			entry.inputs[dt .. "Aegis"] = output[dt .. "Aegis"]
			entry.inputs[dt .. "GuardAbsorb"] = output[dt .. "GuardAbsorb"]
			entry.inputs[dt .. "GuardAbsorbRate"] = output[dt .. "GuardAbsorbRate"]
		end

		local damage = {}
		local any = false
		for _, dt in ipairs(dmgTypeList) do
			local d = output[dt .. "TakenHit"]
			if d and d > 0 then
				damage[dt] = d
				any = true
			end
		end
		entry.damageIn = {}
		for k, v in pairs(damage) do
			entry.damageIn[k] = v
		end

		if any then
			entry.oneHit = poolsOf(calcs.reducePoolsByDamage(nil, damage, actor))
			local damage2 = {}
			for _, dt in ipairs(dmgTypeList) do
				local d = output[dt .. "TakenHit"]
				if d and d > 0 then
					damage2[dt] = d
				end
			end
			local first = calcs.reducePoolsByDamage(nil, damage2, actor)
			local damage3 = {}
			for _, dt in ipairs(dmgTypeList) do
				local d = output[dt .. "TakenHit"]
				if d and d > 0 then
					damage3[dt] = d
				end
			end
			entry.twoHits = poolsOf(calcs.reducePoolsByDamage(first, damage3, actor))
		else
			entry.noDamage = true
		end
	end)
	if not ok then
		entry.error = tostring(err)
	end

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry, { indent = true }))
	out:close()

	io.stderr:write(string.format("reduce-pools-dump: %s%s\n", file,
		entry.error and ("  ERROR " .. entry.error) or ""))
end

io.stderr:write("reduce-pools-dump: wrote " .. #builds .. " builds to " .. outDir .. "\n")
