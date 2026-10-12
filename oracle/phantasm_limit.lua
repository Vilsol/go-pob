local xmlPath = arg[1]
assert(xmlPath, "usage: phantasm_limit.lua <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv
local output = env.player.output
local modDB = env.player.modDB

print(string.format("BUILD %s", xmlPath))
print(string.format("output.ActivePhantasmLimit = %s   output.ActiveSpectreLimit = %s   output.ActiveGolemLimit = %s",
	tostring(output.ActivePhantasmLimit), tostring(output.ActiveSpectreLimit), tostring(output.ActiveGolemLimit)))

local playerOverride = modDB:Override(nil, "ActivePhantasmLimit")
print(string.format("modDB:Override(nil,'ActivePhantasmLimit') = %s", tostring(playerOverride)))
print(string.format("modDB multiplier ActivePhantasmLimitDoubled = %s",
	tostring(modDB.multipliers["ActivePhantasmLimitDoubled"])))

print("")
print("=== every active skill that summons a LIMITED minion (CalcPerform.lua:1401-1425) ===")
local replicaMax = {}
for _, activeSkill in ipairs(env.player.activeSkillList) do
	local minionList = activeSkill.minionList
	if not activeSkill.skillFlags.disable and minionList and minionList[1] then
		local grantedEffect = activeSkill.activeEffect.grantedEffect
		for _, minionType in ipairs(minionList) do
			local minionData = env.data.minions[minionType]
			if minionData and not minionData.hostile and minionData.limit then
				local sml, cfg = activeSkill.skillModList, activeSkill.skillCfg
				local name = minionData.limit
				local override = modDB:Override(nil, name)
				local base = sml:Sum("BASE", cfg, name)
				local inc = sml:Sum("INC", cfg, name)
				local more = sml:More(cfg, name)
				local val = calcLib.val(sml, name)
				local minionLimitMore = sml:More(cfg, "ActiveMinionLimit")
				local count = math.floor(override or (val * minionLimitMore))
				if not replicaMax[name] or count > replicaMax[name] then
					replicaMax[name] = count
				end
				print(string.format("SKILL %-32s minion=%-22s limit=%s",
					grantedEffect.name, minionType, name))
				local sup = {}
				for _, s2 in ipairs(activeSkill.supportList or {}) do
					sup[#sup+1] = string.format("%s(lvl %s)", s2.grantedEffect.id, tostring(s2.level))
				end
				print(string.format("    supportList = %s", table.concat(sup, ", ")))
				print(string.format("    socketGroup = %s  slot=%s",
					tostring(activeSkill.socketGroup and activeSkill.socketGroup.label),
					tostring(activeSkill.socketGroup and activeSkill.socketGroup.slot)))
				print(string.format("    level=%s  BASE=%s INC=%s MORE=%s  val=%s",
					tostring(activeSkill.activeEffect.level), tostring(base), tostring(inc),
					tostring(more), tostring(val)))
				print(string.format("    More(ActiveMinionLimit)=%s  override=%s  -> count=%s",
					tostring(minionLimitMore), tostring(override), tostring(count)))
				for _, m in ipairs(sml:Tabulate("BASE", cfg, name)) do
					print(string.format("      BASE  %-10s %s", tostring(m.mod.value), tostring(m.mod.source)))
				end
				for _, m in ipairs(sml:Tabulate("INC", cfg, name)) do
					print(string.format("      INC   %-10s %s", tostring(m.mod.value), tostring(m.mod.source)))
				end
				for _, m in ipairs(sml:Tabulate("MORE", cfg, name)) do
					print(string.format("      MORE  %-10s %s", tostring(m.mod.value), tostring(m.mod.source)))
				end
			end
		end
	end
end

print("")
print("=== SELF-CHECK: replica vs the engine's own output ===")
local allVerified = true
local anyChecked = false
for name, replica in pairs(replicaMax) do
	anyChecked = true
	local engine = output[name]
	local ok = engine ~= nil and math.abs(replica - engine) < 1e-9
	allVerified = allVerified and ok
	print(string.format("  %-28s replica=%s  engine=%s  VERIFIED=%s",
		name, tostring(replica), tostring(engine), tostring(ok)))
end
if not anyChecked then
	print("  REFUSING: this build has no minion carrying a limit, so nothing was replicated and")
	print("  nothing was checked. An empty self-check must not read as a pass.")
elseif not allVerified then
	print("  ⚠️ REPLICA DOES NOT REPRODUCE THE ENGINE. Every BASE/INC/MORE above is a property of")
	print("  this probe, not of PoB. Discard them and fix the replica against CalcPerform.lua:1401-1425.")
end
