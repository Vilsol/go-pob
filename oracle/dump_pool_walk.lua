
local selfPath = arg[0]
local buildsDir, outDir, listFile = arg[1], arg[2], arg[3]
assert(buildsDir and outDir, "usage: dump_pool_walk.lua <buildsDir> <outDir> [listFile]")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local dmgTypeList = { "Physical", "Lightning", "Cold", "Fire", "Chaos" }
local maxCalls = 400

local function asObject(t)
	return setmetatable(t, { __jsontype = "object" })
end

local damageTable

local function pools(p)
	if type(p) ~= "table" then
		return nil
	end
	return asObject({
		Life = p.Life,
		Mana = p.Mana,
		EnergyShield = p.EnergyShield,
		Ward = p.Ward,
		OverkillDamage = p.OverkillDamage,
		hitPoolRemaining = p.hitPoolRemaining,
		LifeLossLostOverTime = p.LifeLossLostOverTime,
		LifeBelowHalfLossLostOverTime = p.LifeBelowHalfLossLostOverTime,
		damageTakenThatCanBeRecouped = damageTable(p.damageTakenThatCanBeRecouped),
		aegis = damageTable(p.Aegis),
		aegisShared = p.Aegis and p.Aegis.shared or nil,
		aegisSharedElemental = p.Aegis and p.Aegis.sharedElemental or nil,
		guard = damageTable(p.Guard),
		guardShared = p.Guard and p.Guard.shared or nil,
	})
end

damageTable = function(d)
	if type(d) ~= "table" then
		return nil
	end
	local out = {}
	for _, dt in ipairs(dmgTypeList) do
		if d[dt] and d[dt] ~= 0 then
			out[dt] = d[dt]
		end
	end
	return asObject(out)
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

local record, callCount

local function wrapReduce(calcsTab)
	local realReduce = calcsTab.calcs.reducePoolsByDamage
	assert(type(realReduce) == "function", "REFUSING: build.calcsTab.calcs.reducePoolsByDamage is "
		.. "not a function — the name this probe wraps has moved, and wrapping nothing would record "
		.. "an empty walk that reads exactly like a build with no hits")
	calcsTab.calcs.reducePoolsByDamage = function(poolTable, damage, actor)
		local inDamage = damageTable(damage)
		local inPools = pools(poolTable)
		local caller = debug.getinfo(2, "Sl")
		local grandparent = debug.getinfo(3, "Sl")
		local result = realReduce(poolTable, damage, actor)
		if record then
			callCount = callCount + 1
			if #record < maxCalls then
				record[#record + 1] = {
					n = callCount,
					damageIn = inDamage,
					poolsIn = inPools,
					poolsOut = pools(result),
					callerLine = caller and caller.currentline or nil,
					callerSource = caller and caller.short_src or nil,
					cycles = type(damage) == "table" and damage["cycles"] or nil,
					walkCallerLine = grandparent and grandparent.currentline or nil,
				}
			end
		end
		return result
	end
end

for _, file in ipairs(builds) do
	local path = buildsDir .. "/" .. file
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	record, callCount = nil, 0

	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		wrapReduce(build.calcsTab)
		record, callCount = {}, 0
		build.buildFlag = true
		runCallback("OnFrame")

		local output = build.calcsTab.mainEnv.player.output
		entry.inputs = {
			Life = output.Life,
			LifeRecoverable = output.LifeRecoverable,
			ManaUnreserved = output.ManaUnreserved,
			EnergyShieldRecoveryCap = output.EnergyShieldRecoveryCap,
			Ward = output.Ward,
			preventedLifeLossTotal = output.preventedLifeLossTotal,
			NumberOfDamagingHits = output.NumberOfDamagingHits,
			TotalNumberOfHits = output.TotalNumberOfHits,
			TotalEHP = output.TotalEHP,
			anyRecoup = output.anyRecoup,
		}
		local takenHit = {}
		for _, dt in ipairs(dmgTypeList) do
			takenHit[dt] = output[dt .. "TakenHit"]
		end
		entry.takenHit = asObject(takenHit)
		local recoupOut = {}
		for _, dt in ipairs(dmgTypeList) do
			recoupOut[dt] = output[dt .. "RecoupableDamageTaken"]
		end
		entry.recoupableDamageTaken = asObject(recoupOut)
	end)

	if not ok then
		entry.error = tostring(err)
	end
	entry.callCount = callCount
	entry.truncated = callCount > #record
	entry.calls = record
	record = nil

	local outFh = assert(io.open(outDir .. "/" .. file:gsub("%.xml$", "") .. ".json", "w"))
	outFh:write(json.encode(stable(entry), { indent = false }))
	outFh:close()
	ConPrintf("pool-walk: %s (%d calls, %d recorded)", file, callCount, #entry.calls)
end
