local outPath = assert(arg[1], "usage: custom_mod_browser.lua <out.json> [search...]")
local searches = { select(2, unpack(arg)) }
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local list
local controlsOf
main.OpenPopup = function(_, _, _, _, controls)
	controlsOf = controls
end
local configTab = build.configTab
if configTab.OpenAddModPopup then
	configTab:OpenAddModPopup({ text = "" })
else
	require("Modules.ConfigModBrowser").OpenAddModPopup(configTab, { text = "" })
end
assert(controlsOf, "REFUSING: OpenAddModPopup opened no popup")
controlsOf.search:SetText("")
list = controlsOf.listControl.list
assert(#list > 1000, "REFUSING: the Mod Browser lists only " .. #list .. " lines")

local out = { pobVersion = tostring(launch.versionNumber or "unknown"), lines = { } }
local function entryOf(line)
	if type(line) == "table" then
		return { text = line.text, sources = line.sources }
	end
	return line
end
local function textOf(line)
	return type(line) == "table" and line.text or line
end
for i, line in ipairs(list) do
	out.lines[i] = entryOf(line)
end
if #searches > 0 then
	out.searches = { }
	for _, search in ipairs(searches) do
		controlsOf.search:SetText(search, true)
		local shown = { }
		for i, line in ipairs(controlsOf.listControl.list) do
			shown[i] = textOf(line)
		end
		out.searches[#out.searches + 1] = { search = search, shown = shown }
	end
end
local file = assert(io.open(outPath, "w"))
file:write(json.encode(out))
file:close()
