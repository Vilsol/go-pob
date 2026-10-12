
local buildPath, outPath = arg[1], arg[2]
assert(buildPath and outPath, "usage: mirage_dot_terms.lua <build.xml> <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scalars(t)
	local out = {}
	if type(t) ~= "table" then
		return nil
	end
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local dmgTypeList = { "Physical", "Lightning", "Cold", "Fire", "Chaos" }
local isElemental = { Lightning = true, Cold = true, Fire = true }

local function round(val, places)
	local mult = 10 ^ (places or 0)
	return math.floor(val * mult + 0.5) / mult
end

local fh = assert(io.open(buildPath, "r"))
local xmlText = fh:read("*a")
fh:close()

local name = buildPath:match("([^/]+)$")

wipeGlobalCache()
loadBuildFromXML(xmlText, name)
runCallback("OnFrame")

local firstPass = build.calcsTab.mainEnv.player.mainSkill
local firstMirageOutput = firstPass and firstPass.mirage and firstPass.mirage.output

local captured = {}
local calcsMod = build.calcsTab.calcs
local origCopy = calcsMod.copyActiveSkill
calcsMod.copyActiveSkill = function(env, mode, skill)
	local newSkill, newEnv = origCopy(env, mode, skill)
	captured[#captured + 1] = { skill = newSkill, env = newEnv }
	return newSkill, newEnv
end

wipeGlobalCache()
build.calcsTab:BuildOutput()

local mainEnv = build.calcsTab.mainEnv
local entry = { build = name, copyActiveSkillCalls = #captured }

local function terms(skill, env, output)
	local out = { dot = {}, mods = {} }
	if not skill then
		return nil
	end
	local skillModList = skill.skillModList
	local ae = skill.activeEffect
	out.gem = ae and {
		level = ae.level,
		quality = ae.quality,
		effectiveLevel = ae.grantedEffectLevel and ae.grantedEffectLevel.level,
		levelRequirement = ae.grantedEffectLevel and ae.grantedEffectLevel.levelRequirement,
		name = ae.grantedEffect and ae.grantedEffect.name,
		srcLevel = ae.srcInstance and ae.srcInstance.level,
		srcQuality = ae.srcInstance and ae.srcInstance.quality,
		gelValue1 = ae.grantedEffectLevel and ae.grantedEffectLevel[1],
		gelDamageEffectiveness = ae.grantedEffectLevel and ae.grantedEffectLevel.damageEffectiveness,
		gelBaseMultiplier = ae.grantedEffectLevel and ae.grantedEffectLevel.baseMultiplier,
		srcGem = ae.srcInstance and ae.srcInstance.gemData and ae.srcInstance.gemData.name,
		srcNameSpec = ae.srcInstance and ae.srcInstance.nameSpec,
		srcQualityId = ae.srcInstance and ae.srcInstance.qualityId,
	}
	out.skillData = scalars(skill.skillData)
	out.AverageDamage = output.AverageDamage
	out.TotalDPS = output.TotalDPS
	out.TotalDot = output.TotalDot
	out.TotalDotInstance = output.TotalDotInstance
	out.Speed = output.Speed
	out.Duration = output.Duration

	out.mods.DamageMoreSkillCfg = skillModList:More(skill.skillCfg, "Damage")
	out.mods.SpeedMoreSkillCfg = skillModList:More(skill.skillCfg, "Speed")
	out.mods.MirageArcherLessDamage = skillModList:Sum("BASE", skill.skillCfg, "MirageArcherLessDamage")
	out.mods.MirageArcherLessAttackSpeed = skillModList:Sum("BASE", skill.skillCfg, "MirageArcherLessAttackSpeed")
	out.mods.MirageArcherMaxCount = skillModList:Sum("BASE", skill.skillCfg, "MirageArcherMaxCount")

	if skill.dotCfg then
		out.dotCfgFlags = skill.dotCfg.flags
		out.dotCfgKeywordFlags = skill.dotCfg.keywordFlags
		out.dotCfgSkillName = skill.dotCfg.skillName
		out.dotCfgSkillPart = skill.dotCfg.skillPart
		out.dotCfgHasSkillTypes = skill.dotCfg.skillTypes ~= nil
		out.mods.DamageMoreDotCfg = skillModList:More(skill.dotCfg, "Damage")
	end

	for _, damageType in ipairs(dmgTypeList) do
		local cfg = skill["dot" .. damageType .. "Cfg"]
		local base = skill.skillData[damageType .. "Dot"]
		if cfg and (base or (output[damageType .. "Dot"] or 0) > 0) then
			local inc = skillModList:Sum("INC", cfg, "Damage", damageType .. "Damage",
				isElemental[damageType] and "ElementalDamage" or nil)
			local more = round(skillModList:More(cfg, "Damage", damageType .. "Damage",
				isElemental[damageType] and "ElementalDamage" or nil), 2)
			local mult = skillModList:Sum("BASE", cfg, "DotMultiplier", damageType .. "DotMultiplier")
			out.dot[damageType] = {
				baseVal = base or 0,
				inc = inc,
				more = more,
				mult = mult,
				effMult = output[damageType .. "DotEffMult"],
				value = output[damageType .. "Dot"],
				dpsMultiplier = skill.skillData.dpsMultiplier,
			}
		end
	end
	return out
end

local playerSkill = mainEnv.player.mainSkill
if not playerSkill then
	entry.error = "build has no main skill"
elseif not playerSkill.mirage then
	entry.error = "main skill has no mirage table: not triggeredByMirageArcher, or "
		.. "CalcPerform.lua:2443 found no usable skill"
elseif #captured == 0 then
	entry.error = "copyActiveSkill was never called -- the hook did not fire, so nothing below "
		.. "is a measurement of PoB's mirage"
else
	entry.mainSkill = playerSkill.activeEffect.grantedEffect.name
	entry.mirageCount = playerSkill.mirage.count
	entry.player = terms(playerSkill, mainEnv, mainEnv.player.output)

	local mirageCap
	for _, c in ipairs(captured) do
		if c.env.player.output == playerSkill.mirage.output then
			mirageCap = c
		end
	end
	if not mirageCap then
		entry.error = "copyActiveSkill fired " .. #captured .. " time(s) and none of their envs "
			.. "owns the output table CalcPerform.lua:2478 linked into mainSkill.mirage -- the "
			.. "captures are FullDPS or Saviour copies, not the graded mirage"
	else
		entry.mirage = terms(mirageCap.skill, mirageCap.env, mirageCap.env.player.output)
		entry.mirageOutputIsSameTable = mirageCap.env.player.output == playerSkill.mirage.output

		local drift = {}
		if firstMirageOutput then
			for k, v in pairs(firstMirageOutput) do
				if type(v) == "number" then
					local now = playerSkill.mirage.output[k]
					if type(now) ~= "number" or (v ~= now and math.abs(v - now) > math.abs(v) * 1e-9) then
						drift[k] = { firstPass = v, secondPass = now }
					end
				end
			end
		end
		entry.rebuildDrift = drift
		entry.rebuildVerified = firstMirageOutput ~= nil and next(drift) == nil
	end
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(entry, { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. name .. " -> " .. outPath
	.. (entry.error and (" (" .. entry.error .. ")") or "") .. "\n")
