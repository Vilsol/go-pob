
local buildPath, outPath = arg[1], arg[2]
assert(buildPath and outPath, "usage: mirage_modlist.lua <build.xml> <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local bor, band, bnot = bit.bor, bit.band, bit.bnot

local fh = assert(io.open(buildPath, "r"))
local xmlText = fh:read("*a")
fh:close()

local name = buildPath:match("([^/]+)$")

wipeGlobalCache()
loadBuildFromXML(xmlText, name)
runCallback("OnFrame")

local entry = { build = name }
local env = build.calcsTab.mainEnv
local calcs = build.calcsTab.calcs
local skill = env and env.player and env.player.mainSkill

local function fail(why)
	entry.error = why
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry, { indent = true }))
	out:close()
	io.stderr:write("oracle: " .. name .. " -> " .. outPath .. " (" .. why .. ")\n")
	os.exit(0)
end

if not skill then
	fail("build has no main skill")
end
if not skill.skillData.triggeredByMirageArcher then
	fail("main skill is not supported by Mirage Archer, so there is no copy to inspect")
end
if not skill.mirage or not skill.mirage.output then
	fail("PoB built no mirage table for this build (infoMessage2 = "
		.. tostring(skill.infoMessage2) .. ")")
end

local function dotCfgOf(activeSkill)
	local skillCfg, skillData = activeSkill.skillCfg, activeSkill.skillData
	local cfg = {
		skillName = skillCfg.skillName,
		skillPart = skillCfg.skillPart,
		skillTypes = skillCfg.skillTypes,
		slotName = skillCfg.slotName,
		flags = bor(ModFlag.Dot, skillCfg.flags),
		keywordFlags = band(skillCfg.keywordFlags, bnot(KeywordFlag.Hit)),
	}
	for flagName, dataKey in pairs({ Area = "dotIsArea", Projectile = "dotIsProjectile",
		Spell = "dotIsSpell", Attack = "dotIsAttack" }) do
		local flag = ModFlag[flagName]
		if bor(cfg.flags, flag) == cfg.flags and not skillData[dataKey] then
			cfg.flags = band(cfg.flags, bnot(flag))
		end
	end
	return cfg
end

local function damageMods(activeSkill)
	local list = {}
	for _, mod in ipairs(activeSkill.skillModList:List(nil, "Damage")) do
		t_insert(list, modLib.formatMod(mod))
	end
	return list
end

local function probe(label, activeSkill)
	local hitCfg = activeSkill.skillCfg
	local dotCfg = dotCfgOf(activeSkill)
	return {
		label = label,
		moreDamageHit = activeSkill.skillModList:More(hitCfg, "Damage"),
		moreDamageDot = activeSkill.skillModList:More(dotCfg, "Damage"),
		moreSpeedHit = activeSkill.skillModList:More(hitCfg, "Speed"),
		hitFlags = hitCfg.flags,
		hitKeywordFlags = hitCfg.keywordFlags,
		dotFlags = dotCfg.flags,
		dotKeywordFlags = dotCfg.keywordFlags,
		damageMods = damageMods(activeSkill),
	}
end

local usedSkill
if os.getenv("USEDSKILL") == "live" then
	usedSkill = skill
	entry.usedSkillArm = "live env.player.mainSkill (:2431)"
else
	local uuid = cacheSkillUUID(skill)
	if not GlobalCache.cachedData["MAIN"][uuid] then
		calcs.buildActiveSkill(env, "MAIN", skill, true)
	end
	local cached = GlobalCache.cachedData["MAIN"][uuid]
	if not cached or not cached.ActiveSkill then
		fail("GlobalCache holds no ActiveSkill for uuid " .. tostring(uuid))
	end
	usedSkill = cached.ActiveSkill
	entry.usedSkillArm = "GlobalCache copy (:2439)"
end
local moreDamage = usedSkill.skillModList:Sum("BASE", usedSkill.skillCfg, "MirageArcherLessDamage")
local moreAttackSpeed = usedSkill.skillModList:Sum("BASE", usedSkill.skillCfg, "MirageArcherLessAttackSpeed")
local mirageCount = usedSkill.skillModList:Sum("BASE", skill.skillCfg, "MirageArcherMaxCount")

entry.moreDamage = moreDamage
entry.moreAttackSpeed = moreAttackSpeed
entry.mirageCount = mirageCount

local newSkill, newEnv = calcs.copyActiveSkill(env, "MAIN", usedSkill)
newSkill.skillModList:NewMod("Damage", "MORE", moreDamage, "Mirage Archer",
	skill.ModFlags, skill.KeywordFlags)
newSkill.skillModList:NewMod("Speed", "MORE", moreAttackSpeed, "Mirage Archer",
	skill.ModFlags, skill.KeywordFlags)
newEnv.player.mainSkill = newSkill
newSkill.marked = true
newEnv.dontCache = true
calcs.perform(newEnv)

local function close(a, b)
	a, b = a or 0, b or 0
	if a == b then
		return true
	end
	return b ~= 0 and math.abs((a - b) / b) < 1e-9
end

entry.replicaOutput = {
	TotalDPS = newEnv.player.output.TotalDPS,
	TotalDot = newEnv.player.output.TotalDot,
	AverageDamage = newEnv.player.output.AverageDamage,
}
entry.engineOutput = {
	TotalDPS = skill.mirage.output.TotalDPS,
	TotalDot = skill.mirage.output.TotalDot,
	AverageDamage = skill.mirage.output.AverageDamage,
}
entry.verified = close(entry.replicaOutput.TotalDPS, entry.engineOutput.TotalDPS)
	and close(entry.replicaOutput.TotalDot, entry.engineOutput.TotalDot)
	and close(entry.replicaOutput.AverageDamage, entry.engineOutput.AverageDamage)

entry.player = probe("player", skill)
entry.mirage = probe("mirage", newSkill)

if not entry.verified then
	entry.error = "REPLICA DOES NOT REPRODUCE THE ENGINE'S MIRAGE — every More() below is a "
		.. "property of this probe, not of PoB. Do not use them."
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(entry, { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. name .. " -> " .. outPath
	.. " verified=" .. tostring(entry.verified) .. "\n")
