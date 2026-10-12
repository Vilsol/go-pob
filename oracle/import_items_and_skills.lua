local outPath, charPath, buildPath = arg[1], arg[2], arg[3]
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

local function snapshot()
	local items = {}
	for _, id in ipairs(build.itemsTab.itemOrderList) do
		items[#items + 1] = { id = id, raw = build.itemsTab.items[id].raw }
	end
	local slots = {}
	for name, slot in pairs(build.itemsTab.slots) do
		if slot.selItemId and slot.selItemId ~= 0 then
			slots[name] = slot.selItemId
		end
	end
	local groups = {}
	for _, group in ipairs(build.skillsTab.socketGroupList) do
		local gems = {}
		for _, gem in ipairs(group.gemList) do
			gems[#gems + 1] = { nameSpec = gem.nameSpec, level = gem.level, quality = gem.quality, enabled = gem.enabled,
				skillId = gem.skillId or "" }
		end
		groups[#groups + 1] = { label = group.label, enabled = group.enabled, slot = group.slot or "",
			imbuedSupport = group.imbuedSupport or "", gems = gems }
	end
	return { items = items, slots = slots, groups = groups, mainSocketGroup = build.mainSocketGroup,
		level = build.characterLevel }
end

newBuild()
build.characterLevelAutoMode = false
build.importTab:ImportItemsAndSkills(charData, true, true, false)
runCallback("OnFrame")
local first = snapshot()
build.importTab:ImportItemsAndSkills(charData, true, true, false)
runCallback("OnFrame")
local second = snapshot()
build.importTab:ImportItemsAndSkills(charData, false, false, false)
runCallback("OnFrame")
local third = snapshot()

local onto
if buildPath then
	local bh = assert(io.open(buildPath, "r"))
	local xmlText = bh:read("*a")
	bh:close()
	loadBuildFromXML(xmlText, "onto")
	runCallback("OnFrame")
	build.characterLevelAutoMode = false
	build.importTab:ImportItemsAndSkills(charData, true, true, false)
	runCallback("OnFrame")
	onto = snapshot()
end

local gemForBaseName = {}
for name, gemId in pairs(data.gemForBaseName) do
	gemForBaseName[name] = data.gems[gemId].grantedEffectId
end
local vaalGemFor = {}
for grantedEffectId, gemId in pairs(data.gemGrantedEffectIdForVaalGemId) do
	vaalGemFor[grantedEffectId] = data.gems[gemId] and data.gems[gemId].name or ""
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode({ first = first, second = second, third = third, onto = onto, gemForBaseName = gemForBaseName, vaalGemFor = vaalGemFor },
	{ indent = true }))
f:close()
io.stderr:write(string.format("import-items-and-skills: %d items, %d groups\n", #first.items, #first.groups))
