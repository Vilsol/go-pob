
local inPath, outPath = arg[1], arg[2]
assert(inPath and outPath, "usage: rich_notes_pob_load.lua <build.xml> <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local fh = assert(io.open(inPath, "r"))
local xmlText = fh:read("*a")
fh:close()

loadBuildFromXML(xmlText, "rich-notes")
runCallback("OnFrame")

local notes = build.notesTab.controls.edit.buf
local saved = build:SaveDB("rich-notes")

local savedNotes
for _, node in ipairs(common.xml.ParseXML(saved)[1]) do
	if type(node) == "table" and node.elem == "Notes" then
		for _, child in ipairs(node) do
			if type(child) == "string" then
				savedNotes = child
			end
		end
	end
end

local customMods = { }
local configTab = build.configTab
for _, block in ipairs(configTab.configSets[configTab.activeConfigSetId].customModsList) do
	table.insert(customMods, { title = block.title, text = block.text })
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode({
	notes = notes,
	savedHasRichNotes = saved:find("RichNotes", 1, true) ~= nil,
	savedNotes = savedNotes,
	firstSocketGroupLabel = build.skillsTab.socketGroupList[1].label,
	enemyMods = build.partyTab.controls.enemyMods.buf,
	customMods = customMods,
}, { indent = true, keyorder = { "notes", "savedHasRichNotes", "savedNotes", "firstSocketGroupLabel", "enemyMods", "customMods", "title", "text" } }))
out:close()
