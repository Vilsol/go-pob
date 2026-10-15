local outPath, corpus = arg[1], arg[2]
assert(outPath and corpus, "usage: timeless_search.lua <out.json> <corpus dir>")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local cases = {
	{ name = "lp-single", build = "0005-occultist-vaal-double-strike.xml", type = 2, socket = "jewel",
		list = "karui_notable_add_strength, 1\nkarui_notable_add_life, 2, 0, 1\ntotal_strength, 0.5" },
	{ name = "lp-required", build = "0033-saboteur-blade-trap-of-greatswords-2.xml", type = 2, socket = "jewel",
		list = "karui_notable_add_physical_damage, required\nkarui_notable_add_life, 1", fallback = "karui_notable_add_armour, 3, 0, 0\nkarui_notable_add_life, 9", minTotal = 2 },
	{ name = "gv-rolls", build = "0007-elementalist-storm-burst-of-repulsion.xml", type = 1, socket = "jewel",
		list = "vaal_small_fire_damage, 1\nvaal_notable_armour_1, 1, 1\nvaal_small_life, 0.5\nvaal_notable_cold_damage_1, 2" },
	{ name = "mf-protect", build = "0028-occultist-forbidden-rite-of-soul-sacrifice.xml", type = 4, socket = "jewel", conqueror = 3, devotion = { 3, 7 }, realm = 2, tradeType = 3,
		list = "templar_notable_minimum_power_charge, 2\ntotal_devotion, 1\ntemplar_notable_fire_conversion, 1, 0, 1", filter = true, distance = 3, protect = 2 },
	{ name = "mf-plain", build = "0049-hierophant-spark-of-the-nova-2.xml", type = 4, socket = "jewel", devotion = { 5, 5 }, searchMore = true,
		list = "total_devotion, 1\ntemplar_notable_arcane_surge, 4" },
	{ name = "mf-efficiency", build = "0049-hierophant-spark-of-the-nova-2.xml", type = 4, socket = "jewel", devotion = { 15, 17 },
		list = "total_devotion, 1" },
	{ name = "br-all", build = "0005-occultist-vaal-double-strike.xml", type = 3, socket = -1,
		list = "maraketh_notable_add_dexterity, 1\nmaraketh_notable_add_movement_speed, 3\ntotal_dexterity, 0.2", minTotal = 40 },
	{ name = "eh-filter", build = "0005-occultist-vaal-double-strike.xml", type = 5, socket = 61834, filter = true, distance = 4, conqueror = 2, highlight = { 3, 9 }, realm = 3,
		list = "eternal_notable_crit_2, 1\neternal_notable_power_2, 2\neternal_notable_cast_speed_1, 1" },
	{ name = "ht-single", build = "0010-elementalist-storm-burst-2.xml", type = 6, socket = 54127,
		list = "kalguur_notable_1, 2\nkalguur_notable_10, 1\nkalguuran_small_ward, 0.5" },
}

local function load(file)
	local fh = assert(io.open(corpus .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	loadBuildFromXML(xmlText, file)
	runCallback("OnFrame")
end

local jewelNames = { "Glorious Vanity", "Lethal Pride", "Brutal Restraint", "Militant Faith", "Elegant Hubris", "Heroic Tragedy" }

local out = {}
for _, c in ipairs(cases) do
	load(c.build)
	local socket = c.socket
	if socket == "jewel" then
		local ids = {}
		for nodeId, itemId in pairs(build.spec.jewels) do
			local item = build.itemsTab.items[itemId]
			if item and item.name:find(jewelNames[c.type], 1, true) then
				ids[#ids + 1] = nodeId
			end
		end
		table.sort(ids)
		assert(ids[1], "REFUSING: " .. c.build .. " has no " .. jewelNames[c.type])
		socket = ids[1]
	end
	local td = build.timelessData
	td.jewelType = { id = c.type }
	td.conquerorType = { id = c.conqueror or 1 }
	td.jewelSocket = { id = socket }
	td.searchList = c.list
	td.searchListFallback = c.fallback
	td.socketFilter = c.filter or nil
	td.socketFilterDistance = c.distance or 0
	td.totalMinimumWeight = c.minTotal
	td.devotionVariant1 = c.devotion and c.devotion[1] or 1
	td.devotionVariant2 = c.devotion and c.devotion[2] or 1
	build.treeTab:FindTimelessJewel()
	local controls = main.popups[1].controls
	local protected = {}
	assert(not c.protect or controls.protectAllocatedBox1, "REFUSING: " .. c.name .. " has no protectable node")
	for i = 1, c.protect or 0 do
		local box = controls["protectAllocatedBox" .. i]
		if not box then
			break
		end
		box.state = true
		box.changeFunc(true)
		protected[#protected + 1] = box.label
	end
	controls.searchButton.onClick()
	local urls = {}
	OpenURL = function(url) urls[#urls + 1] = url end
	controls.searchTradeLeagueSelect:SetList({ "Settlers", "Standard" })
	controls.realmSelection.selIndex = c.realm or 1
	controls.tradeTypeSelection:SetSel(c.tradeType or 1)
	controls.searchMore.state = c.searchMore or false
	if c.highlight then
		controls.searchResults.selIndex = c.highlight[1]
		controls.searchResults.highlightIndex = c.highlight[2]
	end
	if #td.searchResults > 0 then
		controls.searchTradeButton.onClick()
		controls.searchTradeButton.onClick()
	end
	local itemRaw = ""
	if td.searchResults[1] then
		local addItem = build.itemsTab.AddItem
		build.itemsTab.AddItem = function(tab, item, ...)
			itemRaw = item.raw
			return addItem(tab, item, ...)
		end
		controls.searchResults:OnSelClick(1, td.searchResults[1], true)
		build.itemsTab.AddItem = addItem
	end
	local sockets = {}
	for _, s in ipairs(controls.socketSelect.list) do
		sockets[#sockets + 1] = { id = s.id, label = s.label, keystone = s.keystone }
	end
	local mods = {}
	for _, m in ipairs(controls.nodeSelect.list) do
		mods[#mods + 1] = { id = m.id or "", label = m.label }
	end
	local results, rows = {}, {}
	for _, r in ipairs(td.searchResults) do
		local nodes = {}
		for legionId in pairs(td.sharedResults.desiredNodes or {}) do
			if r[legionId] then
				local names = {}
				for _, n in ipairs(r[legionId].targetNodeNames or {}) do
					names[#names + 1] = n
				end
				table.sort(names)
				nodes[legionId] = names
			end
		end
		results[#results + 1] = { r.seed, r.total, r.socketId or socket }
		if r.seed % 53 == 0 then
			rows[#rows + 1] = { seed = r.seed, socket = r.socketId or socket, label = r.label, nodes = nodes }
		end
	end
	local desired = {}
	for legionId, d in pairs(td.sharedResults.desiredNodes or {}) do
		desired[#desired + 1] = { id = legionId, name = d.displayName, idx = d.desiredIdx }
	end
	table.sort(desired, function(a, b) return a.idx < b.idx end)
	out[#out + 1] = { name = c.name, build = c.build, type = c.type, conqueror = c.conqueror or 1, socket = socket,
		list = c.list, fallback = c.fallback or "", filter = c.filter or false, distance = c.distance or 0,
		minTotal = c.minTotal, protected = protected, sockets = sockets, mods = mods, desired = desired, results = results, rows = rows,
		urls = urls, item = itemRaw, devotion = c.devotion or { 1, 1 }, realm = c.realm or 1, tradeType = c.tradeType or 1,
		searchMore = c.searchMore or false, highlight = c.highlight or { 0, 0 } }
	io.stderr:write(string.format("timeless-search: %s socket %s, %d results\n", c.name, tostring(socket), #results))
	main:ClosePopup()
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out))
f:close()
