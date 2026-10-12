local xmlPath = arg[1]
assert(xmlPath, "usage: gem_level_chain.lua <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv

print(string.format("BUILD %s", xmlPath))
print(string.format("mainSocketGroup=%s characterLevel=%s",
	tostring(env.mainSocketGroup), tostring(build.characterLevel)))

for _, activeSkill in ipairs(env.player.activeSkillList) do
	local ae = activeSkill.activeEffect
	local ge = ae.grantedEffect
	local raw = ae.srcInstance and ae.srcInstance.level
	local isMain = activeSkill == env.player.mainSkill
	print("")
	print(string.format("SKILL %-28s main=%s slot=%s group=%s",
		tostring(ge and ge.id), tostring(isMain),
		tostring(activeSkill.slotName), tostring(activeSkill.socketGroup and activeSkill.socketGroup.label)))
	print(string.format("    raw srcInstance.level=%s -> activeEffect.level=%s  levelRequirement=%s",
		tostring(raw), tostring(ae.level),
		tostring(ae.grantedEffectLevel and ae.grantedEffectLevel.levelRequirement)))

	local sum = 0
	for _, m in ipairs(ae.gemPropertyInfo or {}) do
		local v = m.value
		if v and v.key == "level" then
			sum = sum + (v.value or 0)
			print(string.format("      gemProperty  +%-3s keyword=%-22s source=%s",
				tostring(v.value), tostring(v.keyword or (v.keywordList and table.concat(v.keywordList, "/"))),
				tostring(m.mod and m.mod.source)))
		end
	end
	if activeSkill.skillModList then
		for _, sp in ipairs(activeSkill.skillModList:Tabulate("LIST", activeSkill.skillCfg, "SupportedGemProperty")) do
			local v = sp.value
			if v.keyword == "grants_active_skill" and v.key == "level" and ae.gemData and not ae.gemData.tags.support then
				sum = sum + (v.value or 0)
				print(string.format("      support      +%-3s source=%s", tostring(v.value), tostring(sp.mod and sp.mod.source)))
			end
		end
	end
	if raw then
		local verified = (raw + sum) == ae.level
		print(string.format("    VERIFIED=%s  raw %s + contributors %s = %s (engine says %s)",
			tostring(verified), tostring(raw), tostring(sum), tostring(raw + sum), tostring(ae.level)))
	end


	for _, se in ipairs(activeSkill.effectList or {}) do
		if se.grantedEffect and se.grantedEffect.support then
			local sraw = se.srcInstance and se.srcInstance.level
			print(string.format("      SUPPORT %-28s raw=%s eff=%s", tostring(se.grantedEffect.id), tostring(sraw), tostring(se.level)))
			for _, m in ipairs(se.gemPropertyInfo or {}) do
				local v = m.value
				if v and v.key == "level" then
					print(string.format("          gemProperty +%-3s keyword=%-20s source=%s",
						tostring(v.value), tostring(v.keyword or (v.keywordList and table.concat(v.keywordList, "/"))),
						tostring(m.mod and m.mod.source)))
				end
			end
		end
	end

	if activeSkill.minion then
		print(string.format("    minion type=%s level=%s   minionLevelIsPlayerLevel=%s minionLevel=%s minionLevelIsEnemyLevel=%s",
			tostring(activeSkill.minion.type), tostring(activeSkill.minion.level),
			tostring(activeSkill.skillData.minionLevelIsPlayerLevel),
			tostring(activeSkill.skillData.minionLevel),
			tostring(activeSkill.skillData.minionLevelIsEnemyLevel)))
	end
end
