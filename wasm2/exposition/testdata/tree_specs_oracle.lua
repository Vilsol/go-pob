-- Loads a build and edits its passive trees through the tree list's and the top bar's own controls, one step at a
-- time, recalculating after each. Prints one JSON row per step: the step name, the active spec, the tree list's
-- labels, every spec as Save writes it, and a few output stats.
-- usage: tree_specs_oracle.lua <out.jsonl> <build.xml> specs|class
local outPath, xmlPath, scenario = arg[1], arg[2], arg[3]
assert(outPath and xmlPath and (scenario == "specs" or scenario == "class"), "usage: tree_specs_oracle.lua <out.jsonl> <build.xml> specs|class")

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. tostring(s):gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeNumber(v)
	return string.format("%.17g", v)
end

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()
loadBuildFromXML(xmlText, "tree_specs")

local build = __mainObject__.main.modes.BUILD
local tab = build.treeTab

-- a popup's text box takes the step's title and its Save is pressed; a confirm popup takes the step's answer
local title, answer
main.OpenPopup = function(_, _, _, _, controls)
	controls.edit:SetText(title, true)
	controls.save.onClick()
end
main.OpenConfirmPopup = function(_, _, _, _, onConfirm, _, onExtra)
	if answer == "continue" then
		onConfirm()
	elseif answer == "connect" then
		onExtra()
	end
end

local list = new("PassiveSpecListControl"):PassiveSpecListControl(nil, { 0, 50, 350, 200 }, tab)

local stats = { "Life", "Str", "Dex", "Int" }
local out = assert(io.open(outPath, "w"))
local function row(step)
	build.buildFlag = true
	runCallback("OnFrame")
	local labels = {}
	for i, l in ipairs(tab:GetSpecList()) do
		labels[i] = encodeString(l)
	end
	local specs = {}
	for i, spec in ipairs(tab.specList) do
		local xml = { elem = "Spec" }
		spec:Save(xml)
		local a = xml.attrib
		local fields = {}
		for _, k in ipairs({ "title", "treeVersion", "classId", "ascendClassId", "secondaryAscendClassId", "nodes", "masteryEffects" }) do
			fields[#fields + 1] = encodeString(k) .. ":" .. encodeString(a[k] == nil and "" or a[k])
		end
		for _, child in ipairs(xml) do
			if child.elem == "WeaponSet1" or child.elem == "WeaponSet2" then
				fields[#fields + 1] = encodeString(child.elem) .. ":" .. encodeString(child.attrib.nodes)
			elseif child.elem == "Sockets" then
				local socks = {}
				for _, s in ipairs(child) do
					socks[#socks + 1] = s.attrib.nodeId .. ":" .. s.attrib.itemId
				end
				fields[#fields + 1] = '"sockets":' .. encodeString(table.concat(socks, ","))
			elseif child.elem == "Overrides" then
				for _, o in ipairs(child) do
					fields[#fields + 1] = '"overrides":' .. encodeString(o.attrib.strNodes .. "|" .. o.attrib.dexNodes .. "|" .. o.attrib.intNodes)
				end
			end
		end
		specs[i] = "{" .. table.concat(fields, ",") .. "}"
	end
	local parts = {}
	for _, stat in ipairs(stats) do
		local v = build.calcsTab.mainOutput[stat]
		if type(v) == "number" then
			parts[#parts + 1] = encodeString(stat) .. ":" .. encodeNumber(v)
		end
	end
	out:write('{"step":', encodeString(step), ',"active":', tab.activeSpec, ',"className":', encodeString(build.spec.curClassName),
		',"ascendClassName":', encodeString(build.spec.curAscendClassName), ',"labels":[', table.concat(labels, ","), '],"specs":[',
		table.concat(specs, ","), '],"output":{', table.concat(parts, ","), "}}\n")
end

local function select(index)
	build.modFlag = true
	tab:SetActiveSpec(index)
end
local function pick(index)
	list.selIndex = index
	list.selValue = list.list[index]
end
-- ListControl.lua:440-448: a drag that drops the picked row at to
local function move(from, to)
	pick(from)
	local value = list.selValue
	table.remove(list.list, from)
	table.insert(list.list, to, value)
	list:OnOrderChange(from, to)
end
local function classDrop(label, choice)
	answer = choice
	for i, v in ipairs(build.controls.classDrop.list) do
		if v.label == label then
			build.controls.classDrop.selFunc(i, v)
		end
	end
	answer = nil
end
local function ascendDrop(ascendClassId)
	for i, v in ipairs(build.controls.classDrop:GetSelValueByKey("ascendancies") or {}) do
		if v.ascendClassId == ascendClassId then
			build.controls.ascendDrop.selFunc(i, v)
		end
	end
end

row("load")

if scenario == "specs" then
	title = "Second"
	list.controls.new.onClick()
	row("new Second")
	pick(1)
	title = "Radius copy"
	list.controls.copy.onClick()
	row("copy 1 Radius copy")
	pick(2)
	title = "Renamed"
	list.controls.rename.onClick()
	row("rename 2 Renamed")
	select(4)
	row("select 4")
	move(4, 1)
	row("move 4 1")
	move(2, 3)
	row("move 2 3")
	answer = "continue"
	pick(3)
	list:OnSelDelete(3, list.selValue)
	row("delete 3")
	pick(1)
	list:OnSelDelete(1, list.selValue)
	row("delete 1")
	select(2)
	row("select 2")
	title = "Third"
	list.controls.new.onClick()
	row("new Third")
	pick(2)
	list:OnSelDelete(2, list.selValue)
	row("delete 2")
else
	ascendDrop(1)
	row("ascendancy 1")
	ascendDrop(0)
	row("ascendancy 0")
	classDrop("Sorceress", nil)
	row("class Sorceress cancel")
	classDrop("Sorceress", "connect")
	row("class Sorceress connect")
	classDrop("Ranger", "continue")
	row("class Ranger continue")
	classDrop("Witch", nil)
	row("class Witch")
end
out:close()
