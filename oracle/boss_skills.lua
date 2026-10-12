
local outPath = arg[1]
assert(outPath, "usage: boss_skills.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

assert(data.bossSkills, "REFUSING: the pinned PoB does not expose data.bossSkills")

local skillFields = {
	DamageType = true,
	DamageMultipliers = true,
	UberDamageMultiplier = true,
	DamagePenetrations = true,
	UberDamagePenetrations = true,
	speed = true,
	UberSpeed = true,
	critChance = true,
	earlierUber = true,
	additionalStats = true,
	tooltip = true,
}

local unknown = {}

local skills = {}
local skillCount, multCount, penCount = 0, 0, 0

local function penValue(where, v)
	if v == "" then
		return json.null
	end
	assert(type(v) == "number", string.format(
		"REFUSING: %s is neither a number nor the empty string (%s)", where, tostring(v)))
	return v
end

local function penTable(where, tbl)
	if not tbl then
		return nil
	end
	local out = {}
	for penType, pen in pairs(tbl) do
		out[tostring(penType)] = penValue(where .. "." .. tostring(penType), pen)
		penCount = penCount + 1
	end
	return out
end

local statCount = 0
local function statTable(where, tbl)
	if not tbl then
		return nil
	end
	local out = {}
	for k, v in pairs(tbl) do
		if tostring(v) == "flag" then
			out[tostring(k)] = "flag"
		else
			assert(type(v) == "number", string.format(
				"REFUSING: %s.%s is neither a number nor \"flag\" (%s)", where, tostring(k), tostring(v)))
			out[tostring(k)] = v
		end
		statCount = statCount + 1
	end
	return out
end

local function additionalStatsTable(where, tbl)
	if not tbl then
		return nil
	end
	for k in pairs(tbl) do
		assert(k == "base" or k == "uber", string.format(
			"REFUSING: %s carries %q; only base and uber are modelled", where, tostring(k)))
	end
	return {
		base = statTable(where .. ".base", tbl.base),
		uber = statTable(where .. ".uber", tbl.uber),
	}
end

for name, skill in pairs(data.bossSkills) do
	for k in pairs(skill) do
		if not skillFields[k] then
			unknown[#unknown + 1] = "bossSkills." .. tostring(name) .. "." .. tostring(k)
		end
	end

	local mults = {}
	for damageType, damageMult in pairs(skill.DamageMultipliers or {}) do
		assert(#damageMult == 2, string.format(
			"REFUSING: bossSkills.%s.DamageMultipliers.%s has %d elements, not 2",
			tostring(name), tostring(damageType), #damageMult))
		mults[tostring(damageType)] = { damageMult[1], damageMult[2] }
		multCount = multCount + 1
	end

	skills[tostring(name)] = {
		DamageType = skill.DamageType,
		DamageMultipliers = mults,
		UberDamageMultiplier = skill.UberDamageMultiplier,
		additionalStats = additionalStatsTable("bossSkills." .. tostring(name) .. ".additionalStats",
			skill.additionalStats),
		DamagePenetrations = penTable("bossSkills." .. tostring(name) .. ".DamagePenetrations",
			skill.DamagePenetrations),
		UberDamagePenetrations = penTable(
			"bossSkills." .. tostring(name) .. ".UberDamagePenetrations", skill.UberDamagePenetrations),
		speed = skill.speed,
		UberSpeed = skill.UberSpeed,
		critChance = skill.critChance,
		earlierUber = skill.earlierUber and true or false,
	}
	skillCount = skillCount + 1
end

if #unknown > 0 then
	io.stderr:write("REFUSING: PoB carries fields this dumper does not model:\n")
	for _, f in ipairs(unknown) do
		io.stderr:write("  " .. f .. "\n")
	end
	io.stderr:write("Emitting anyway would silently drop them from data/boss_skills_generated.go, " ..
		"and nothing downstream would notice. Model them here and in boss_skills_emit.py.\n")
	os.exit(3)
end

assert(skillCount > 0, "REFUSING: data.bossSkills dumped zero presets")

local out = {
	pobVersion = tostring(launch.versionNumber or "unknown"),
	bossSkills = skills,
}

local file = assert(io.open(outPath, "w"), "REFUSING: cannot write " .. outPath)
file:write(json.encode(out, { indent = true }))
file:close()

io.stderr:write(string.format(
	"wrote %s (PoB %s, %d presets, %d damage multipliers, %d penetrations)\n",
	outPath, out.pobVersion, skillCount, multCount, penCount))
