local dir = arg[1]
dofile("HeadlessWrapper.lua")
local t_insert = table.insert
local files = { "0160-elementalist-winter-orb-10.xml", "0121-assassin-poisonous-concoction-of-bouncing-2.xml", "0356-elementalist-winter-orb-24.xml", "0354-gladiator-vitality-2.xml", "0388-luminary-flame-link.xml", "0341-necromancer-kinetic-blast-of-clustering.xml" }

local function str(v)
	local s = tostring(v):gsub('\\', '\\\\'):gsub('"', '\\"'):gsub('\n', '\\n'):gsub('\t', '\\t'):gsub('\r', '\\r')
	return '"' .. s .. '"'
end
local function read(name)
	local f = assert(io.open(dir .. "/" .. name, "rb"))
	local text = f:read("*a")
	f:close()
	return text
end
local function xmlEscape(v)
	return (v:gsub("&", "&amp;"):gsub("<", "&lt;"):gsub(">", "&gt;"))
end
local function partyXML(text)
	local dbXML = common.xml.ParseXML(text)
	local parts = { "<PathOfBuilding><Party>" }
	for _, tabNode in ipairs(dbXML[1]) do
		if type(tabNode) == "table" and tabNode.elem == "Party" then
			for _, node in ipairs(tabNode) do
				if type(node) == "table" and node.elem == "ExportedBuffs" then
					t_insert(parts, '<ExportedBuffs name="' .. xmlEscape(node.attrib.name) .. '">' .. xmlEscape(node[1] or "") .. "</ExportedBuffs>")
				end
			end
			break
		end
	end
	t_insert(parts, "</Party></PathOfBuilding>")
	return table.concat(parts)
end
local function exported(text)
	local dbXML = common.xml.ParseXML(text)
	for _, tabNode in ipairs(dbXML[1]) do
		if type(tabNode) == "table" and tabNode.elem == "Party" then
			local out = { }
			for _, node in ipairs(tabNode) do
				if type(node) == "table" and node.elem == "ExportedBuffs" then
					t_insert(out, { elem = "ImportedBuffs", attrib = { name = node.attrib.name }, node[1] })
				end
			end
			return out
		end
	end
	return { }
end

local labels = { Aura = "simpleAuras", Curse = "simpleCurses", ["Warcry Skills"] = "simpleWarcries", ["Link Skills"] = "simpleLinks", EnemyMods = "simpleEnemyMods" }
local bufs = { PartyMemberStats = "editPartyMemberStats", Aura = "editAuras", Curse = "editCurses", ["Warcry Skills"] = "editWarcries", ["Link Skills"] = "editLinks", EnemyConditions = "enemyCond", EnemyMods = "enemyMods" }
local names = { "PartyMemberStats", "Aura", "Curse", "Warcry Skills", "Link Skills", "EnemyConditions", "EnemyMods" }

local function dumpBuffers(tab)
	local parts = { }
	for _, name in ipairs(names) do
		t_insert(parts, str(name) .. ":" .. str(tab.controls[bufs[name]].buf))
	end
	return "{" .. table.concat(parts, ",") .. "}"
end

local summaries = { }
for _, file in ipairs(files) do
	newBuild()
	local tab = build.partyTab
	local xml = exported(read(file))
	xml.attrib = { }
	tab:Load(xml)
	local parts = { }
	for _, name in ipairs(names) do
		if labels[name] then
			t_insert(parts, str(name) .. ":" .. str(tab.controls[labels[name]].label))
		end
	end
	t_insert(summaries, "{" .. str("file") .. ":" .. str(file) .. "," .. str("buffers") .. ":" .. dumpBuffers(tab) .. "," .. str("labels") .. ":{" .. table.concat(parts, ",") .. "}}")
end

local scenarios = {
	{ name = "replace all", steps = { { import = files[1], dest = "All", append = false } } },
	{ name = "replace then append all", steps = { { import = files[1], dest = "All", append = false }, { import = files[2], dest = "All", append = true } } },
	{ name = "append onto itself", steps = { { import = files[1], dest = "All", append = false }, { import = files[1], dest = "All", append = true } } },
	{ name = "append curse only", steps = { { import = files[2], dest = "All", append = false }, { import = files[1], dest = "Curse", append = true } } },
	{ name = "replace enemy mods only", steps = { { import = files[1], dest = "All", append = false }, { import = files[3], dest = "EnemyMods", append = false } } },
	{ name = "stats only", steps = { { import = files[1], dest = "Party Member Stats", append = false } } },
	{ name = "clear auras", steps = { { import = files[2], dest = "All", append = false }, { clear = "Aura" } } },
	{ name = "append links then warcries", steps = { { import = files[5], dest = "All", append = false }, { import = files[4], dest = "All", append = true }, { import = files[6], dest = "Aura", append = true } } },
	{ name = "clear all", steps = { { import = files[1], dest = "All", append = false }, { clear = "All" } } },
}

local out = { }
for _, sc in ipairs(scenarios) do
	newBuild()
	local tab = build.partyTab
	local steps = { }
	for _, step in ipairs(sc.steps) do
		if step.clear then
			tab.controls.importCodeDestination:SelByValue(step.clear)
			tab.controls.clear.onClick()
			t_insert(steps, "{" .. str("clear") .. ":" .. str(step.clear) .. "}")
		else
			tab.controls.importCodeDestination:SelByValue(step.dest)
			tab.controls.appendNotReplace.state = step.append
			tab.importCodeSite = nil
			tab.importCodeXML = partyXML(read(step.import))
			tab.importCodeValid = true
			tab.controls.importCodeGo.onClick()
			t_insert(steps, "{" .. str("import") .. ":" .. str(step.import) .. "," .. str("dest") .. ":" .. str(step.dest) .. "," .. str("append") .. ":" .. tostring(step.append) .. "}")
		end
	end
	local saved = { }
	tab:Save(saved)
	local nodes = { }
	for _, node in ipairs(saved) do
		t_insert(nodes, "[" .. str(node.elem) .. "," .. str(node.attrib.name) .. "," .. str(node[1]) .. "]")
	end
	local attrib = "{" .. str("destination") .. ":" .. str(saved.attrib.destination) .. "," .. str("append") .. ":" .. str(saved.attrib.append) .. "," .. str("ShowAdvanceTools") .. ":" .. str(saved.attrib.ShowAdvanceTools) .. "}"
	t_insert(out, "{" .. str("name") .. ":" .. str(sc.name) .. "," .. str("steps") .. ":[" .. table.concat(steps, ",") .. "]," .. str("buffers") .. ":" .. dumpBuffers(tab) .. "," .. str("saved") .. ":[" .. table.concat(nodes, ",") .. "]," .. str("attrib") .. ":" .. attrib .. "}")
end

local inputs = { }
for _, file in ipairs(files) do
	t_insert(inputs, str(file) .. ":" .. str(partyXML(read(file))))
end
print("{" .. str("inputs") .. ":{" .. table.concat(inputs, ",") .. "}," .. str("summaries") .. ":[\n" .. table.concat(summaries, ",\n") .. "\n]," .. str("scenarios") .. ":[\n" .. table.concat(out, ",\n") .. "\n]}")
