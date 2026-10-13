local outPath, charPath = arg[1], arg[2]
if not pcall(require, "zlib") then
	package.preload.zlib = function()
		local refuse = function() error("lua-zlib is not installed (mise run bootstrap in the PoB checkout)") end
		return { deflate = refuse, inflate = refuse }
	end
end
if not pcall(require, "lua-utf8") then
	package.preload["lua-utf8"] = function() return string end
end

dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local fh = assert(io.open(charPath, "r"))
local charData = json.decode(fh:read("*a")).character
fh:close()

local function sorted(list)
	table.sort(list)
	return list
end

local function snapshot(message)
	local spec = build.spec
	local nodes = {}
	for id in pairs(spec.allocNodes) do
		nodes[#nodes + 1] = id
	end
	local masteries = {}
	for node, effect in pairs(spec.masterySelections) do
		masteries[#masteries + 1] = node .. ":" .. effect
	end
	local sockets = {}
	for node, item in pairs(spec.jewels) do
		if item > 0 then
			sockets[#sockets + 1] = node .. ":" .. item
		end
	end
	local overrides = {}
	for node, override in pairs(spec.hashOverrides or {}) do
		overrides[#overrides + 1] = node .. ":" .. tostring(override.dn)
	end
	local items = {}
	for _, id in ipairs(build.itemsTab.itemOrderList) do
		items[#items + 1] = { id = id, raw = build.itemsTab.items[id].raw }
	end
	local input = build.configTab.input
	return {
		message = message,
		classId = spec.curClassId, ascendClassId = spec.curAscendClassId,
		secondaryAscendClassId = spec.curSecondaryAscendClassId, treeVersion = spec.treeVersion,
		clusterHashFormatVersion = spec.clusterHashFormatVersion,
		nodes = sorted(nodes), masteries = sorted(masteries), sockets = sorted(sockets), overrides = sorted(overrides),
		items = items,
		level = build.characterLevel, levelAuto = build.characterLevelAutoMode,
		bandit = input.bandit, pantheonMajorGod = input.pantheonMajorGod, pantheonMinorGod = input.pantheonMinorGod,
		resistancePenalty = input.resistancePenalty,
	}
end

newBuild()
local first = snapshot(build.importTab:ImportPassiveTreeAndJewels(charData, true))
runCallback("OnFrame")
local second = snapshot(build.importTab:ImportPassiveTreeAndJewels(charData, false))
runCallback("OnFrame")
local third = snapshot(build.importTab:ImportPassiveTreeAndJewels(charData, true))

local f = assert(io.open(outPath, "w"))
f:write(json.encode({ first = first, second = second, third = third }, { indent = true }))
f:close()
io.stderr:write(string.format("import-passive-tree: %d nodes, %d sockets, %d items\n", #first.nodes, #first.sockets, #first.items))
