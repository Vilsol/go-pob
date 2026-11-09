-- usage (from PoB-PoE2's src/): luajit poe2_party_import.lua <out.json> <giver.xml> <buffers.txt>
-- Records PoB-PoE2's party tab text boxes and saved Party element after a series of import-code imports and clears
-- (PartyTab.lua's importCodeGo and clear buttons), over two exports: the giver build's own (Export support on) and a
-- hand-written set of buffers, each under a "=== <name>" line.
local outPath, giverPath, buffersPath = arg[1], arg[2], arg[3]
assert(outPath and giverPath and buffersPath, "usage: poe2_party_import.lua <out.json> <giver.xml> <buffers.txt>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function read(path)
	local f = assert(io.open(path))
	local text = f:read("*a")
	f:close()
	return text
end
local function xmlEscape(v)
	return (v:gsub("&", "&amp;"):gsub("<", "&lt;"):gsub(">", "&gt;"))
end
local function partyXML(nodes)
	local parts = { '<?xml version="1.0" encoding="UTF-8"?>\n<PathOfBuilding2><Build className="Warrior" ascendClassName="None" level="80"/><Party>' }
	for _, node in ipairs(nodes) do
		parts[#parts + 1] = '<ExportedBuffs name="' .. xmlEscape(node.name) .. '">' .. xmlEscape(node.text) .. "</ExportedBuffs>"
	end
	parts[#parts + 1] = "</Party></PathOfBuilding2>"
	return table.concat(parts)
end

loadBuildFromXML(read(giverPath))
build.partyTab.enableExportBuffs = true
build.buildFlag = true
runCallback("OnFrame")
local saved = {}
build.partyTab:Save(saved)
local giverNodes = {}
for _, node in ipairs(saved) do
	if node.elem == "ExportedBuffs" then
		giverNodes[#giverNodes + 1] = { name = node.attrib.name, text = node[1] }
	end
end
local bufferNodes, current = {}, nil
for line in (read(buffersPath) .. "\n"):gmatch("([^\n]*)\n") do
	local header = line:match("^=== (.+)$")
	if header then
		current = { name = header }
		bufferNodes[#bufferNodes + 1] = current
	elseif current then
		current.text = current.text and (current.text .. "\n" .. line) or line
	end
end
for _, node in ipairs(bufferNodes) do
	node.text = node.text:gsub("\n+$", "")
end
local inputs = { giver = partyXML(giverNodes), buffers = partyXML(bufferNodes) }

local bufs = { PartyMemberStats = "editPartyMemberStats", Aura = "editAuras", Curse = "editCurses", ["Warcry Skills"] = "editWarcries", ["Link Skills"] = "editLinks", EnemyConditions = "enemyCond", EnemyMods = "enemyMods" }

local scenarios = {
	{ name = "replace all", steps = { { import = "giver", dest = "All", append = false } } },
	{ name = "replace then append all", steps = { { import = "giver", dest = "All", append = false }, { import = "buffers", dest = "All", append = true } } },
	{ name = "append onto itself", steps = { { import = "buffers", dest = "All", append = false }, { import = "buffers", dest = "All", append = true } } },
	{ name = "append curse only", steps = { { import = "buffers", dest = "All", append = false }, { import = "giver", dest = "Curse", append = true } } },
	{ name = "replace enemy mods only", steps = { { import = "giver", dest = "All", append = false }, { import = "buffers", dest = "EnemyMods", append = false } } },
	{ name = "stats only", steps = { { import = "buffers", dest = "Party Member Stats", append = false } } },
	{ name = "clear auras", steps = { { import = "buffers", dest = "All", append = false }, { clear = "Aura" } } },
	{ name = "append links then warcries", steps = { { import = "buffers", dest = "Link Skills", append = false }, { import = "buffers", dest = "Link Skills", append = true }, { import = "giver", dest = "Warcry Skills", append = true } } },
	{ name = "clear all", steps = { { import = "buffers", dest = "All", append = false }, { clear = "All" } } },
}

local out = { inputs = inputs, scenarios = {} }
for _, sc in ipairs(scenarios) do
	newBuild()
	local tab = build.partyTab
	for _, step in ipairs(sc.steps) do
		if step.clear then
			tab.controls.importCodeDestination:SelByValue(step.clear)
			tab.controls.clear.onClick()
		else
			tab.controls.importCodeDestination:SelByValue(step.dest)
			tab.controls.appendNotReplace.state = step.append
			tab.importCodeSite = nil
			tab.importCodeXML = inputs[step.import]
			tab.importCodeValid = true
			tab.controls.importCodeGo.onClick()
		end
	end
	local buffers = {}
	for name, control in pairs(bufs) do
		buffers[name] = tab.controls[control].buf
	end
	local saved = {}
	tab:Save(saved)
	local nodes = {}
	for _, node in ipairs(saved) do
		nodes[#nodes + 1] = { elem = node.elem, name = node.attrib.name, text = node[1] }
	end
	out.scenarios[#out.scenarios + 1] = { name = sc.name, steps = sc.steps, buffers = buffers, saved = nodes, attrib = saved.attrib }
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode(out, { indent = true, keyorder = { "name", "steps", "buffers", "saved", "attrib", "inputs", "scenarios" } }))
f:write("\n")
f:close()
