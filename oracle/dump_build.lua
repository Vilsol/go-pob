
local buildsDir, outDir = arg[1], arg[2]
assert(buildsDir and outDir, "usage: dump_build.lua <buildsDir> <outDir>")

local selfDir = arg[0]:match("^(.*)[/\\]") or "."
local construct = dofile(selfDir .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
dofile(selfDir .. "/apply_range_guard.lua").install()


if os.getenv("FORCE_CLUSTER_KEY") == "1" then
	local realParseMod = modLib.parseMod
	modLib.parseMod = function(line, isComb)
		local modList, extra = realParseMod(line, isComb)
		if type(modList) == "table" then
			for _, m in ipairs(modList) do
				if m.name == "JewelData" and m.type == "LIST" and type(m.value) == "table"
					and m.value.key == "clusterJewelSkill"
					and m.value.value == "affliction_curse_effect_small" then
					m.value.value = "affliction_curse_effect"
				end
			end
		end
		return modList, extra
	end
	io.stderr:write("dump: FORCE_CLUSTER_KEY=1 (Medium affliction_curse_effect forced)\n")
end

local emptyModCacheMode = os.getenv("EMPTY_MODCACHE")
local function emptyModCache()
	for k in pairs(modLib.parseModCache) do
		modLib.parseModCache[k] = nil
	end
end
if emptyModCacheMode == "1" or emptyModCacheMode == "each" then
	local n = 0
	for _ in pairs(modLib.parseModCache) do
		n = n + 1
	end
	emptyModCache()

	local rebuilt = 0
	for version in pairs(main.tree) do
		main.tree[version] = construct("PassiveTree", version)
		rebuilt = rebuilt + 1
	end

	io.stderr:write(string.format(
		"dump: EMPTY_MODCACHE=%s (dropped %d shipped cache keys, rebuilt %d passive tree(s))\n",
		emptyModCacheMode, n, rebuilt))
end

local staticFromItem = {}
local staticFromItemCount = 0
for id, granted in pairs(data.skills) do
	if granted.fromItem then
		staticFromItem[id] = true
		staticFromItemCount = staticFromItemCount + 1
	end
end

if staticFromItemCount == 0 then
	io.stderr:write("dump: REFUSING -- data.skills carries no static fromItem rows, so the " ..
		"snapshot ran before the data was loaded and the per-build reset would be a no-op\n")
	os.exit(1)
end

local function resetFromItem()
	for id, granted in pairs(data.skills) do
		if staticFromItem[id] then
			granted.fromItem = true
		else
			granted.fromItem = nil
		end
	end
end

io.stderr:write(string.format(
	"dump: fromItem reset armed (%d rows carry it statically and are preserved)\n",
	staticFromItemCount))

local keystoneSource = {}
local keystoneSourceCount = 0
local snapshottedTrees = {}
local function snapshotKeystoneSources(tree)
	if not tree or snapshottedTrees[tree] then
		return
	end
	snapshottedTrees[tree] = true
	local function record(mod)
		if keystoneSource[mod] == nil then
			keystoneSource[mod] = mod.source or false
			keystoneSourceCount = keystoneSourceCount + 1
		end
	end
	for _, node in pairs(tree.keystoneMap or {}) do
		for _, mod in ipairs(node.modList or {}) do
			record(mod)
			if type(mod.value) == "table" and mod.value.mod then
				record(mod.value.mod)
			end
		end
	end
end
for _, tree in pairs(main.tree) do
	snapshotKeystoneSources(tree)
end
local loadTree = main.LoadTree
function main:LoadTree(treeVersion)
	local tree = loadTree(self, treeVersion)
	snapshotKeystoneSources(tree)
	return tree
end

if keystoneSourceCount == 0 then
	io.stderr:write("dump: REFUSING -- no loaded passive tree carries keystone mods, so the " ..
		"keystone source snapshot ran before the tree was built\n")
	os.exit(1)
end

local function resetKeystoneSources()
	for mod, source in pairs(keystoneSource) do
		mod.source = source or nil
	end
end

io.stderr:write(string.format(
	"dump: keystone source reset armed (%d keystone mods snapshotted at boot)\n",
	keystoneSourceCount))

local levelCost = {}
local levelCostCount = 0
for _, granted in pairs(data.skills) do
	for _, level in pairs(granted.levels or {}) do
		if type(level) == "table" then
			levelCost[level] = level.cost or false
			levelCostCount = levelCostCount + 1
		end
	end
end

if levelCostCount == 0 then
	io.stderr:write("dump: REFUSING -- data.skills carries no levels, so the cost snapshot ran " ..
		"before the data was loaded and the per-build reset would be a no-op\n")
	os.exit(1)
end

local function resetLevelCosts()
	for level, cost in pairs(levelCost) do
		level.cost = cost or nil
	end
end

io.stderr:write(string.format(
	"dump: skill level cost reset armed (%d levels snapshotted at boot)\n", levelCostCount))

local json = require("dkjson")
local lfs = require("lfs")

local FLAG_NAMES = _G.ModFlag
local KW_NAMES = _G.KeywordFlag

local function stripColor(s)
	if type(s) ~= "string" then
		return tostring(s)
	end
	s = s:gsub("%^%x%x%x%x%x%x", "")
	s = s:gsub("%^%d", "")
	return s
end

local function enrichSource(spec, src)
	if type(src) ~= "string" then
		return src
	end
	local nodeId = src:match("^Tree:(%d+)")
	if nodeId and spec and spec.nodes then
		local node = spec.nodes[tonumber(nodeId)]
		if node and node.dn then
			return src .. " (" .. node.dn .. ")"
		end
	end
	return src
end

local function safeFormat(fn, ...)
	local ok, res = pcall(fn, ...)
	if ok then
		return res
	end
	return "<format error>"
end

local function dumpValue(v)
	local t = type(v)
	if t == "number" or t == "boolean" or t == "string" then
		return v
	end
	return safeFormat(modLib.formatValue, v)
end

local function dumpTags(mod)
	local out = {}
	for _, tag in ipairs(mod) do
		out[#out + 1] = tag.type
	end
	if #out == 0 then
		return nil
	end
	return out
end

local function dumpMod(spec, mod)
	return {
		type = mod.type,
		value = dumpValue(mod.value),
		flags = (mod.flags ~= 0) and safeFormat(modLib.formatFlags, mod.flags, FLAG_NAMES) or nil,
		kw = (mod.keywordFlags ~= 0) and safeFormat(modLib.formatFlags, mod.keywordFlags, KW_NAMES) or nil,
		source = enrichSource(spec, mod.source),
		tags = dumpTags(mod),
		fmt = safeFormat(modLib.formatMod, mod),
	}
end

local function dumpModDB(modDB, spec)
	local out = {}
	if type(modDB) ~= "table" or type(modDB.mods) ~= "table" then
		return nil
	end
	for name, list in pairs(modDB.mods) do
		local mods = {}
		for _, mod in ipairs(list) do
			mods[#mods + 1] = dumpMod(spec, mod)
		end
		out[name] = mods
	end
	return out
end

local function dumpModList(modList, spec)
	local out = {}
	if type(modList) ~= "table" then
		return nil
	end
	for _, mod in ipairs(modList) do
		local name = mod.name
		if name then
			out[name] = out[name] or {}
			table.insert(out[name], dumpMod(spec, mod))
		end
	end
	return out
end

local TAB_TYPES = { "BASE", "INC", "MORE", "OVERRIDE", "FLAG", "MAX" }

local function statSources(modDB, spec, stat)
	local entry = {}
	local any = false
	for _, mt in ipairs(TAB_TYPES) do
		local rows = modDB:Tabulate(mt, nil, stat)
		if #rows > 0 then
			local list = {}
			for _, r in ipairs(rows) do
				list[#list + 1] = {
					source = enrichSource(spec, r.mod.source),
					value = dumpValue(r.value),
					fmt = safeFormat(modLib.formatMod, r.mod),
				}
			end
			entry[mt] = list
			any = true
		end
	end
	if not any then
		return nil
	end
	return entry
end

local function renderBreakdown(bd)
	if type(bd) ~= "table" then
		return { stripColor(bd) }
	end
	local lines = {}
	for _, v in ipairs(bd) do
		if type(v) == "string" or type(v) == "number" then
			lines[#lines + 1] = stripColor(v)
		elseif type(v) == "table" and (v.base or v.total) then
			lines[#lines + 1] = string.format("  %s: base=%s%s%s total=%s",
				tostring(v.sourceName or v.source or "?"),
				tostring(v.base or ""), tostring(v.inc or ""), tostring(v.more or ""), tostring(v.total or ""))
		end
	end
	if bd.slots then
		for _, s in ipairs(bd.slots) do
			lines[#lines + 1] = string.format("  slot %s: base=%s%s%s total=%s",
				tostring(s.sourceName or s.source or "?"),
				tostring(s.base or ""), tostring(s.inc or ""), tostring(s.more or ""), tostring(s.total or ""))
		end
	end
	if bd.rowList and bd.colList then
		local cols = {}
		for _, c in ipairs(bd.colList) do
			cols[#cols + 1] = c.key
		end
		for _, row in ipairs(bd.rowList) do
			local parts = {}
			for _, k in ipairs(cols) do
				if row[k] ~= nil then
					parts[#parts + 1] = k .. "=" .. stripColor(tostring(row[k]))
				end
			end
			if #parts > 0 then
				lines[#lines + 1] = "  " .. table.concat(parts, " ")
			end
		end
	end
	return lines
end

local function dumpBreakdowns(bdTable)
	if type(bdTable) ~= "table" then
		return nil
	end
	local out = {}
	for stat, bd in pairs(bdTable) do
		if type(stat) == "string" then
			local lines = renderBreakdown(bd)
			if #lines > 0 then
				out[stat] = lines
			end
		end
	end
	return out
end

local function scalars(t)
	local out = {}
	if type(t) ~= "table" then
		return out
	end
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local function keyUnion(a, b)
	local seen = {}
	for k in pairs(a) do
		if type(k) == "string" then
			seen[k] = true
		end
	end
	for k in pairs(b) do
		if type(k) == "string" then
			seen[k] = true
		end
	end
	return seen
end

lfs.mkdir(outDir)

local builds = {}
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)

for _, file in ipairs(builds) do
	local path = buildsDir .. "/" .. file
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		wipeGlobalCache()

		resetFromItem()

		resetKeystoneSources()

		resetLevelCosts()

		if emptyModCacheMode == "each" then
			emptyModCache()
		end

		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		local mainEnv = build.calcsTab.mainEnv
		local calcsEnv = build.calcsTab.calcsEnv
		local modDB = mainEnv.modDB
		local spec = mainEnv.spec

		entry.meta = {
			class = build.spec and build.spec.curClassName,
			ascendancy = build.spec and build.spec.curAscendClassName,
			level = build.characterLevel,
			mainSkill = mainEnv.player.mainSkill and mainEnv.player.mainSkill.activeEffect
				and mainEnv.player.mainSkill.activeEffect.grantedEffect
				and mainEnv.player.mainSkill.activeEffect.grantedEffect.name,
		}

		entry.output = scalars(mainEnv.player.output)

		entry.outputMainHand = mainEnv.player.output.MainHand
			and scalars(mainEnv.player.output.MainHand) or nil
		entry.outputOffHand = mainEnv.player.output.OffHand
			and scalars(mainEnv.player.output.OffHand) or nil

		local names = keyUnion(mainEnv.player.output, modDB.mods)
		local sources = {}
		for stat in pairs(names) do
			local s = statSources(modDB, spec, stat)
			if s then
				s.value = mainEnv.player.output[stat]
				sources[stat] = s
			end
		end
		entry.statSources = sources

		entry.modDB = dumpModDB(modDB, spec)
		entry.multipliers = scalars(modDB.multipliers)
		entry.conditions = scalars(modDB.conditions)

		entry.configInput = scalars(mainEnv.configInput)
		entry.configPlaceholder = scalars(mainEnv.configPlaceholder)

		local mainSkill = mainEnv.player.mainSkill
		entry.skillModList = mainSkill and dumpModList(mainSkill.skillModList, spec) or nil
		entry.baseSkillModList = mainSkill and dumpModList(mainSkill.baseSkillModList, spec) or nil
		entry.skillFlags = mainSkill and scalars(mainSkill.skillFlags) or nil
		entry.skillData = mainSkill and scalars(mainSkill.skillData) or nil
		if mainSkill then
			local counts, node, depth = {}, mainSkill.skillModList, 0
			while type(node) == "table" and depth < 8 do
				counts[#counts + 1] = #node
				node = rawget(node, "parent")
				depth = depth + 1
			end
			entry.skillModListChain = counts
			entry.activeEffectMods = mainSkill.activeEffect
				and dumpModList(mainSkill.activeEffect.grantedEffectLevel
					and mainSkill.activeEffect.modList, spec) or nil
			local supports = {}
			for _, se in ipairs(mainSkill.supportList or {}) do
				supports[#supports + 1] = se.grantedEffect and se.grantedEffect.name
			end
			entry.supports = supports
		end

		entry.enemyModDB = mainEnv.enemy and dumpModDB(mainEnv.enemy.modDB, spec) or nil
		entry.enemyOutput = mainEnv.enemy and scalars(mainEnv.enemy.output) or nil

		local minion = mainEnv.minion
		entry.minionOutput = minion and scalars(minion.output) or nil
		entry.minionOutputMainHand = minion and minion.output and minion.output.MainHand
			and scalars(minion.output.MainHand) or nil
		entry.minionOutputOffHand = minion and minion.output and minion.output.OffHand
			and scalars(minion.output.OffHand) or nil
		entry.minionModDB = minion and dumpModDB(minion.modDB, spec) or nil
		entry.minionMultipliers = minion and minion.modDB and scalars(minion.modDB.multipliers) or nil
		entry.minionConditions = minion and minion.modDB and scalars(minion.modDB.conditions) or nil
		local minionSkill = minion and minion.mainSkill
		entry.minionSkillModList = minionSkill and dumpModList(minionSkill.skillModList, spec) or nil
		entry.minionBaseSkillModList = minionSkill and dumpModList(minionSkill.baseSkillModList, spec) or nil
		entry.minionSkillFlags = minionSkill and scalars(minionSkill.skillFlags) or nil
		entry.minionSkillData = minionSkill and scalars(minionSkill.skillData) or nil
		if minionSkill then
			local msupports = {}
			for _, se in ipairs(minionSkill.supportList or {}) do
				msupports[#msupports + 1] = se.grantedEffect and se.grantedEffect.name
			end
			entry.minionSupports = msupports
		end

		entry.minionMeta = minion and {
			name = minion.minionData and minion.minionData.name,
			level = minion.level,
			activeSkill = minion.mainSkill and minion.mainSkill.activeEffect
				and minion.mainSkill.activeEffect.grantedEffect
				and minion.mainSkill.activeEffect.grantedEffect.name,
		} or nil

		entry.breakdown = dumpBreakdowns(calcsEnv and calcsEnv.player.breakdown)
		entry.calcsOutput = calcsEnv and scalars(calcsEnv.player.output) or nil
	end)
	if not ok then
		entry.error = tostring(err)
	end

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry, { indent = true }))
	out:close()

	io.stderr:write(string.format("dump: %s -> %s%s\n", file, outPath,
		entry.error and ("  ERROR " .. entry.error) or ""))
end

io.stderr:write("dump: wrote " .. #builds .. " builds to " .. outDir .. "\n")
