-- Loads a build and edits its socket groups through the Skills tab's own controls, one step at a time, recalculating
-- after each. Prints one JSON row per step: the step name, the main output stats and mainSocketGroup, then the saved
-- <Skills> element after the last step.
-- usage: skills_edit_oracle.lua <out.jsonl> <build.xml> edits|generated
local outPath, xmlPath, scenario = arg[1], arg[2], arg[3]
assert(outPath and xmlPath and (scenario == "edits" or scenario == "generated"), "usage: skills_edit_oracle.lua <out.jsonl> <build.xml> edits|generated")

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
loadBuildFromXML(xmlText, "skills_edit")

local build = __mainObject__.main.modes.BUILD
local tab = build.skillsTab
-- the confirm popups of a delete answer yes
main.OpenConfirmPopup = function(_, _, _, _, onConfirm)
	onConfirm()
end

local stats = { "Life", "Mana", "ManaUnreserved", "Spirit", "TotalDPS", "CombinedDPS", "AverageDamage", "Speed", "ManaCost", "Int" }
local out = assert(io.open(outPath, "w"))
local function row(step)
	build.buildFlag = true
	runCallback("OnFrame")
	local output = build.calcsTab.mainOutput
	local parts = {}
	for _, stat in ipairs(stats) do
		local v = output[stat]
		if type(v) == "number" then
			parts[#parts + 1] = encodeString(stat) .. ":" .. encodeNumber(v)
		end
	end
	local labels = {}
	for i, g in ipairs(tab.socketGroupList) do
		labels[i] = encodeString(g.displayLabel or "")
	end
	out:write('{"step":', encodeString(step), ',"mainSocketGroup":', build.mainSocketGroup, ',"groups":[', table.concat(labels, ","),
		'],"output":{', table.concat(parts, ","), "}}\n")
end

local function select(index)
	local list = tab.controls.groupList
	list.selIndex = index
	list.selValue = tab.socketGroupList[index]
	tab:SetDisplayGroup(tab.socketGroupList[index])
	tab:UpdateGemSlots()
end
local function slot(index)
	tab:UpdateGemSlots()
	return tab.gemSlots[index]
end

row("load")

if scenario == "edits" then

-- SkillListControl.lua:58-71
tab.controls.groupList.controls.new.onClick()
row("new group")
-- SkillListControl.lua:194-198: right click sets the main group
build.mainSocketGroup = 2
tab:AddUndoState()
row("main group 2")

-- SkillsTab.lua:815-875: typing a gem's name picks it
slot(1).nameSpec.gemChangeFunc("Metadata/Items/Gems/SkillGemFireball", true, false, true)
row("add Fireball")
slot(2).nameSpec.gemChangeFunc("Metadata/Items/Gems/SkillGemElementalFocusSupport", true, false, true)
row("add Elemental Focus")

-- SkillsTab.lua:883-898, 909-924, 1027-1047, 1070-1085
slot(1).level.changeFunc("15")
row("Fireball level 15")
slot(1).quality.changeFunc("20")
row("Fireball quality 20")
slot(2).enabled.changeFunc(false)
row("disable Elemental Focus")
slot(2).enabled.changeFunc(true)
row("enable Elemental Focus")
slot(1).count.changeFunc("3")
row("Fireball count 3")

-- SkillsTab.lua:151-156, 202-206, 220-224
tab.controls.groupLabel.changeFunc("Fire")
row("label Fire")
tab.controls.includeInFullDPS.changeFunc(true)
row("include in Full DPS")

tab.controls.groupEnabled.changeFunc(false)
row("disable group")
tab.controls.groupEnabled.changeFunc(true)
row("enable group")

-- ListControl.lua:440-448: drag group 2 above group 1
do
	local list = tab.controls.groupList
	list.selIndex, list.selValue = 2, tab.socketGroupList[2]
	table.remove(list.list, 2)
	table.insert(list.list, 1, list.selValue)
	list:OnOrderChange(2, 1)
end
row("move group 2 to 1")

-- replace the support, then delete the active gem
select(1)
slot(2).nameSpec.gemChangeFunc("Metadata/Items/Gems/SkillGemFrostBomb", true, false, true)
row("replace Elemental Focus with Frost Bomb")
slot(1).delete.onClick()
row("delete Fireball")

-- SkillListControl.lua:139-173
select(2)
tab.controls.groupList:OnSelDelete(2, tab.socketGroupList[2])
row("delete group 2")

elseif scenario == "generated" then
	-- the calculation generates the default attack groups; edits reach them, and a new group goes after them
	tab.controls.groupList.controls.new.onClick()
	local n = #tab.socketGroupList
	row("new group")
	slot(1).nameSpec.gemChangeFunc("Metadata/Items/Gems/SkillGemRollingSlam", true, false, true)
	row("add Rolling Slam")
	build.mainSocketGroup = n
	tab:AddUndoState()
	row("main group new")
	-- SkillsTab.lua:1116-1132
	slot(1).corruptLevel.selFunc(2, slot(1).corruptLevel.list[2])
	row("Rolling Slam +1 level corruption")
	-- SkillsTab.lua:157-171
	tab.controls.set1Enabled.changeFunc(false)
	row("new group set 1 off")
	tab.controls.set2Enabled.changeFunc(false)
	row("new group set 2 off refused")
	select(2)
	slot(2).nameSpec.gemChangeFunc("Metadata/Items/Gems/SkillGemElementalFocusSupport", true, false, true)
	row("support group 2")
	tab.controls.groupCount.changeFunc("2")
	row("group 2 count 2")
	tab.controls.groupEnabled.changeFunc(false)
	row("disable group 2")
	-- SkillListControl.lua:45-54
	tab.controls.groupList.controls.deleteAll.onClick()
	row("delete all")
end

local skills = { elem = "Skills" }
tab:Save(skills)
local xmlOut = common.xml.ComposeXML({ elem = "PathOfBuilding2", { elem = "Build", attrib = { mainSocketGroup = tostring(build.mainSocketGroup) } }, skills })
out:write('{"step":"saved","xml":', encodeString(xmlOut), "}\n")
out:close()
