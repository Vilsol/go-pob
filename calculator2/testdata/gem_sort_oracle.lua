-- Loads a build, opens a gem slot's gem list and lets it sort by DPS, as GemSelectControl does when it drops down.
-- Prints one JSON row: the base DPS, then every gem the sort calculates, with canSupport and the DPS it found.
-- usage: gem_sort_oracle.lua <out.json> <build.xml> <group> <gem slot> <dps field>
local outPath, xmlPath, group, slotIndex, field = arg[1], arg[2], tonumber(arg[3]), tonumber(arg[4]), arg[5]
assert(outPath and xmlPath and group and slotIndex and field, "usage: gem_sort_oracle.lua <out.json> <build.xml> <group> <gem slot> <dps field>")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeNumber(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return '"' .. tostring(v) .. '"'
	end
	return string.format("%.17g", v)
end

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()
loadBuildFromXML(xmlText, "gem_sort")

local build = __mainObject__.main.modes.BUILD
local tab = build.skillsTab
tab.sortGemsByDPSField = field
build.buildFlag = true
runCallback("OnFrame")

tab.controls.groupList.selIndex = group
tab.controls.groupList.selValue = tab.socketGroupList[group]
tab:SetDisplayGroup(tab.socketGroupList[group])
tab:UpdateGemSlots()
local control = tab.gemSlots[slotIndex].nameSpec
-- DPSBuilder yields to the frame loop every 50 ms; a clock that stands still lets it finish in one call
GetTime = function()
	return 0
end
-- GemSelectControl.lua:49-53: the list drops down
control:UpdateSortCache()
control:BuildList("")
control:DPSBuilder()

local sortCache = control.sortCache
local ids = {}
for gemId in pairs(sortCache.dps) do
	local gemData = control.gems[gemId]
	if sortCache.canSupport[gemId] or (gemData.grantedEffect.hasGlobalEffect and not gemData.grantedEffect.support) then
		ids[#ids + 1] = gemId
	end
end
table.sort(ids)
local rows = {}
for _, gemId in ipairs(ids) do
	rows[#rows + 1] = "{" .. '"id":' .. encodeString(control.gems[gemId].id) .. ',"canSupport":' .. tostring(sortCache.canSupport[gemId] == true)
		.. ',"dps":' .. encodeNumber(sortCache.dps[gemId]) .. "}"
end
local out = assert(io.open(outPath, "w"))
out:write('{"base":', encodeNumber(sortCache.baseDPS), ',"gems":[', table.concat(rows, ",\n"), "]}\n")
out:close()
