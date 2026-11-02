-- Loads a build and reads the Set 1 and Set 2 boxes of each socket group as the Skills tab shows them: whether the
-- label is in NEGATIVE, whether the box can be changed, and the group's forcedBoth.
-- usage: weapon_sets_oracle.lua <out.json> <build.xml>
local outPath, xmlPath = arg[1], arg[2]
assert(outPath and xmlPath, "usage: weapon_sets_oracle.lua <out.json> <build.xml>")

dofile("HeadlessWrapper.lua")

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()
loadBuildFromXML(xmlText, "weapon_sets")

local build = __mainObject__.main.modes.BUILD
local tab = build.skillsTab
build.buildFlag = true
runCallback("OnFrame")

local rows = {}
for i, socketGroup in ipairs(tab.socketGroupList) do
	tab:SetDisplayGroup(socketGroup)
	local cells = {}
	for set, control in ipairs({ tab.controls.set1Enabled, tab.controls.set2Enabled }) do
		-- SkillsTab.lua:172-186
		cells[set] = '{"valid":' .. tostring(not control.label():find(colorCodes.NEGATIVE, 1, true)) .. ',"enabled":' .. tostring(control.enabled() and true or false) .. "}"
	end
	rows[i] = '{"group":' .. i .. ',"sets":[' .. table.concat(cells, ",") .. '],"forcedBoth":' .. tostring(socketGroup.forcedBoth == true) .. "}"
end
local out = assert(io.open(outPath, "w"))
out:write("[", table.concat(rows, ",\n"), "]\n")
out:close()
