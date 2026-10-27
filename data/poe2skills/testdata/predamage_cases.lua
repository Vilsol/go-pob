-- Runs every preDamageFunc of PoB-PoE2's loaded data.skills over the cases below and prints what each
-- leaves behind as `case<TAB>key<TAB>value` rows; predamage_expected.tsv is this script's output, which
-- predamage_test.go grades the Go ports against.
dofile("HeadlessWrapper.lua")

local cases = {
	{ name = "DetonatingArrow", id = "DetonatingArrowPlayer", sums = { ["BASE:Multiplier:DetonatingArrowStage"] = 4 } },
	{ name = "Snipe", id = "SnipePlayer", skillData = { channelPercentOfAttackTime = 0.35, hitTimeMultiplier = 9 } },
	{ name = "SnipeNil", id = "SnipePlayer", skillData = { hitTimeMultiplier = 9, channelTimeMultiplier = 9 } },
	{ name = "Bonestorm", id = "BonestormPlayer", sums = { ["BASE:Multiplier:BonestormStage"] = 7 }, output = { ProjectileCount = 3 } },
	{ name = "BonestormZero", id = "BonestormPlayer", sums = { ["BASE:Multiplier:BonestormStage"] = 7 }, output = { ProjectileCount = 0 } },
	{ name = "BonestormNil", id = "BonestormPlayer", sums = { ["BASE:Multiplier:BonestormStage"] = 7 } },
	{ name = "Flameblast", id = "FlameblastPlayer", sums = { ["BASE:Multiplier:FlameblastStage"] = 10 } },
	{ name = "FlickerStrike", id = "FlickerStrikePlayer", sums = { ["BASE:AdditionalFlickersPerPowerCharge"] = 2 } },
	{ name = "GatheringStorm", id = "GatheringStormPlayer", skillData = { channelPercentOfAttackTime = 1.5 } },
	{ name = "Incinerate", id = "IncineratePlayer", sums = { ["BASE:Multiplier:IncinerateStage"] = 3 } },
	{ name = "PlasmaBlast", id = "PlasmaBlastPlayer", skillData = { channelTimeMultiplier = 4 } },
	{ name = "PerfectStrike", id = "PerfectStrikePlayer", skillData = { channelPercentOfAttackTime = 0.8 } },
	{ name = "ShieldCharge", id = "ShieldChargePlayer" },
	{ name = "SuperchargedSlam", id = "SuperchargedSlamPlayer", sums = { ["BASE:Multiplier:SuperchargedSlamStage"] = 5 } },
	{ name = "MinionInstability", id = "MinionInstability", skillData = { selfFireExplosionLifeMultiplier = 0.01 }, output = { Life = 1234 } },
	{ name = "MinionInstabilityNil", id = "MinionInstability", skillData = { selfFireExplosionLifeMultiplier = 0.01 } },
	{ name = "TemperWeapon", id = "TemperWeaponPlayer", sums = { ["BASE:Multiplier:TemperWeaponStage"] = 6 } },
	{ name = "ChaosFrogInactive", id = "ChaosFrogExplosionPlayer" },
	{ name = "ChaosFrog", id = "ChaosFrogExplosionPlayer", sums = { ["BASE:ToadSpawnInterval"] = 750 } },
	{ name = "EnemyExplodeSource", id = "EnemyExplode", statSetIndex = 1, explodeSource = { modSource = "Item:7:Ring" },
		tab = {
			{ source = "Item:7:Ring", value = { type = "RandomElement", amount = 9, value = 25 } },
			{ source = "Tree:55", value = { type = "Fire", amount = 4, value = 100 } },
			{ source = "Item:7:Ring", value = { type = "Cold", amount = 6, value = 40 } },
		} },
	{ name = "EnemyExplodeTree", id = "EnemyExplode", statSetIndex = 2, explodeSource = { id = "55" },
		tab = {
			{ source = "Tree:55", value = { type = "Fire", amount = 4, value = 30 } },
		} },
	{ name = "EnemyExplodeAll", id = "EnemyExplode", statSetIndex = 3,
		list = {
			{ type = "Physical", amount = 10, value = 30 },
			{ type = "Physical", amount = 10, value = 20 },
			{ type = "Fire", amount = 5, value = 50 },
		} },
	{ name = "EnemyExplodeRandom", id = "EnemyExplode", statSetIndex = 3,
		list = {
			{ type = "RandomElement", amount = 12, value = 60 },
		} },
}

local function num(v)
	if v == math.huge then
		return "+Inf"
	elseif v == -math.huge then
		return "-Inf"
	elseif v ~= v then
		return "NaN"
	end
	return string.format("%.17g", v)
end

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys)
	return keys
end

for _, case in ipairs(cases) do
	local ge = data.skills[case.id]
	assert(ge and ge.preDamageFunc, case.id .. " has no preDamageFunc")
	local newMods = {}
	local skillData = {}
	for k, v in pairs(case.skillData or {}) do
		skillData[k] = v
	end
	local output = {}
	for k, v in pairs(case.output or {}) do
		output[k] = v
	end
	local tab = {}
	for i, row in ipairs(case.tab or {}) do
		tab[i] = { value = row.value, mod = { source = row.source } }
	end
	local activeSkill = {
		skillData = skillData,
		skillCfg = {},
		skillModList = {
			Sum = function(_, modType, _, name)
				return (case.sums or {})[modType .. ":" .. name] or 0
			end,
			NewMod = function(_, name, modType, value, source)
				newMods[#newMods + 1] = name .. "|" .. modType .. "|" .. (type(value) == "number" and num(value) or tostring(value)) .. "|" .. tostring(source)
			end,
			List = function()
				return case.list or {}
			end,
			Tabulate = function()
				return tab
			end,
		},
		activeEffect = {
			statSet = { index = case.statSetIndex },
			srcInstance = { explodeSource = case.explodeSource or {} },
		},
	}
	local ok, err = pcall(ge.preDamageFunc, activeSkill, output, {})
	if not ok then
		io.write(case.name, "\terror\t", "nil arithmetic", "\n")
	end
	for _, k in ipairs(sortedKeys(skillData)) do
		io.write(case.name, "\tskillData.", k, "\t", num(skillData[k]), "\n")
	end
	for _, k in ipairs(sortedKeys(output)) do
		io.write(case.name, "\toutput.", k, "\t", num(output[k]), "\n")
	end
	for i, m in ipairs(newMods) do
		io.write(case.name, "\tnewMod.", i, "\t", m, "\n")
	end
	if err and not tostring(err):find("arithmetic") then
		io.write(case.name, "\tunexpected\t", tostring(err), "\n")
	end
end
