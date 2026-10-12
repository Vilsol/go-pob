local xmlPath = arg[1]
assert(xmlPath, "usage: aura_support_absorption.lua <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv

print(string.format("BUILD %s", xmlPath))
print(string.format("mainSocketGroup = %s   activeItemSet.useSecondWeaponSet = %s",
	tostring(env.mainSocketGroup),
	tostring(build.itemsTab.activeItemSet and build.itemsTab.activeItemSet.useSecondWeaponSet)))

print("")
print("=== socketGroupList, as the fill loop at CalcSetup.lua:1502-1506 sees it ===")
for index, group in ipairs(build.skillsTab.socketGroupList) do
	local slot = group.slot and build.itemsTab.slots[group.slot]
	local supports = {}
	for _, gem in ipairs(group.gemList or {}) do
		if gem.gemData and gem.gemData.grantedEffect and gem.gemData.grantedEffect.support then
			supports[#supports + 1] = string.format("%s(lvl %s)", gem.gemData.grantedEffect.id, tostring(gem.level))
		end
	end
	print(string.format("[%2d] slot=%-16s slotName=%-14s source=%-44s enabled=%-5s slotEnabled=%-5s weaponSet=%s supports={%s}",
		index,
		tostring(group.slot),
		tostring(group.slot and group.slot:gsub(" Swap", "")),
		tostring(group.source),
		tostring(group.enabled),
		tostring(group.slotEnabled),
		tostring(slot and slot.weaponSet),
		table.concat(supports, ", ")))
end

local function auraSourceReach(db, source)
	local hits = {}
	for name, mods in pairs(db.mods or {}) do
		for _, m in ipairs(mods) do
			if m.source == source then
				hits[#hits + 1] = string.format("%s=%s", name, tostring(m.value))
			end
		end
	end
	table.sort(hits)
	return table.concat(hits, " ")
end
print("")
print(string.format("PLAYER modDB Skill:Envy -> {%s}", auraSourceReach(env.player.modDB, "Skill:Envy")))
if env.minion then
	print(string.format("MINION modDB Skill:Envy -> {%s}", auraSourceReach(env.minion.modDB, "Skill:Envy")))
end

print("")
print("=== every active skill: its supportList and its aura self gate ===")
for _, activeSkill in ipairs(env.player.activeSkillList) do
	local grantedEffect = activeSkill.activeEffect.grantedEffect
	local sup = {}
	for _, s in ipairs(activeSkill.supportList or {}) do
		sup[#sup + 1] = string.format("%s(lvl %s%s)", s.grantedEffect.id, tostring(s.level),
			s.superseded and ", superseded" or "")
	end
	local sg = activeSkill.socketGroup
	print(string.format("SKILL %-28s source=%-40s slot=%-16s disable=%s level=%s",
		grantedEffect.id, tostring(sg and sg.source), tostring(sg and sg.slot),
		tostring(activeSkill.skillFlags.disable), tostring(activeSkill.activeEffect.level)))
	print(string.format("    supportList = {%s}", table.concat(sup, ", ")))
	for _, s in ipairs(activeSkill.supportList or {}) do
		local ok, res = pcall(calcLib.canGrantedEffectSupportActiveSkill, s.grantedEffect, activeSkill)
		print(string.format("      support %-28s canSupport=%-5s grantedEffect.fromItem=%-5s activeModSource=%s",
			s.grantedEffect.id, ok and tostring(res) or ("ERROR: " .. tostring(res)),
			tostring(s.grantedEffect.fromItem),
			tostring(activeSkill.activeEffect.grantedEffect.modSource)))
	end
	print(string.format("    skillFlags.aura=%s  skillData.auraCannotAffectSelf=%s  buffs=%s",
		tostring(activeSkill.skillFlags.aura),
		tostring(activeSkill.skillData.auraCannotAffectSelf),
		tostring(activeSkill.skillFlags.buffs)))
end
