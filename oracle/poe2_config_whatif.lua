-- usage (from PoB-PoE2's src/): luajit poe2_config_whatif.lua <build.xml> <out.json> [<input xml>...]
-- Records, for a build, the stat comparison lines the config tab's tooltips show: every shown check option toggled
-- (ConfigTab.lua:774-838), every other entry of every shown list option, and each custom modifier group's enable
-- check (ConfigTab.lua:62-79); then, for the build and for each extra input set (<Input> elements added to its active
-- config set), every option control's placeholder after the build loads.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_config_whatif.lua <build.xml> <out.json> [<input xml>...]")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()

local function load(text)
	loadBuildFromXML(text)
	runCallback("OnFrame")
end

-- the lines from the header on, the header's colour code kept
local function compareLines(tooltip)
	local lines, from = {}, nil
	for i, line in ipairs(tooltip.lines) do
		if line.text and line.text:match("will give you:") then
			from = i
		end
		if from and line.text then
			lines[#lines + 1] = line.text
		end
	end
	return lines
end

load(xml)
local tab = build.configTab
local tooltip = new("Tooltip"):Tooltip()

local options = {}
local vars = {}
for var in pairs(tab.varControls) do
	vars[#vars + 1] = var
end
table.sort(vars)
for _, var in ipairs(vars) do
	local control = tab.varControls[var]
	if control:IsShown() and control.tooltipFunc then
		if control.list then
			for i, entry in ipairs(control.list) do
				tooltip:Clear()
				control.tooltipFunc(tooltip, "DROP", i, entry)
				local lines = compareLines(tooltip)
				if #lines > 0 then
					options[#options + 1] = { var = var, value = entry.val, lines = lines }
				end
			end
		elseif control._className == "CheckBoxControl" then
			tooltip:Clear()
			control.tooltipFunc(tooltip)
			local lines = compareLines(tooltip)
			if #lines > 0 then
				options[#options + 1] = { var = var, value = not tab.configSets[tab.activeConfigSetId].input[var], lines = lines }
			end
		end
	end
end

local custom = {}
for i, block in ipairs(tab.customModsBlockControls or {}) do
	local t = new("Tooltip"):Tooltip()
	block.controls.enableCheck.tooltipFunc(t)
	local lines = {}
	for _, line in ipairs(t.lines) do
		if line.text then
			lines[#lines + 1] = line.text
		end
	end
	custom[#custom + 1] = { group = i - 1, lines = lines }
end

local function placeholders()
	local out = {}
	for var, control in pairs(build.configTab.varControls) do
		local n = tonumber(control.placeholder)
		if n then
			out[var] = n
		end
	end
	return out
end

local placeholderSets = { { inputs = "", placeholders = placeholders() } }
for i = 3, #arg do
	local inputs = arg[i]
	load((xml:gsub("(<ConfigSet [^>]*>)", "%1" .. inputs, 1)))
	placeholderSets[#placeholderSets + 1] = { inputs = inputs, placeholders = placeholders() }
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ options = options, custom = custom, placeholders = placeholderSets }), { indent = true }))
o:write("\n")
o:close()
