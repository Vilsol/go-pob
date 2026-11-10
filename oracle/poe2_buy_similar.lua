-- usage (from PoB-PoE2's src/): luajit poe2_buy_similar.lua <build.xml> <out.json>
-- Records CompareBuySimilar.openPopup's popup for every item the build equips, as the Compare tab opens it for a compared
-- item: whether it is unique, its category label, base name and defence rows, and the trade URL the Open URL button
-- copies with the popup as it opens, with every box set, and with every searchable mod row checked, its query decoded,
-- and its mod rows.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_buy_similar.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local buySimilar = LoadModule("Classes/CompareBuySimilar")
-- the popup's mod entries, which it keeps local
local addModEntries = buySimilar.addModEntries
buySimilar.addModEntries = function(...)
	buySimilar.lastModEntries = addModEntries(...)
	return buySimilar.lastModEntries
end
local tradeHelpers = LoadModule("Classes/TradeHelpers")

-- the league list is not fetched: TradeQueryRequests:FetchLeagues falls back to Standard
launch.DownloadPage = function(_, _, callback)
	callback(nil, "offline")
end
local popup, copied
main.OpenPopup = function(_, _, _, _, controls)
	popup = controls
end
Copy = function(text)
	copied = text
end
OpenURL = function() end
-- the headless host has no zlib: the URL carries the query uncompressed, and the Go test gunzips its own
Deflate = function(str)
	return str
end
Inflate = Deflate

local function search()
	-- a control's callback rebuilds the URL, as the league list arriving does
	popup.listedDrop.selFunc(popup.listedDrop.selIndex, popup.listedDrop.list[popup.listedDrop.selIndex])
	copied = nil
	popup.search.onClick()
	local base, encoded = copied:match("^(.*)/([^/]+)$")
	return { base = base, query = json.decode(tradeHelpers.B64GzipDecode(encoded)) }
end

local out = {}
for slotName, slot in pairs(build.itemsTab.slots) do
	local item = build.itemsTab.items[slot.selItemId]
	if item and not slot.nodeId then
		-- the popup remembers the last Listed and league choice; each item opens as the first does
		buySimilar.lastListedIndex, buySimilar.lastLeagueIdx = nil, nil
		buySimilar.openPopup(item, slotName, build)
		local isUnique = item.rarity == "UNIQUE" or item.rarity == "RELIC"
		local rec = { unique = isUnique, name = item.name, baseName = item.baseName, category = tradeHelpers.getTradeCategoryLabel(slotName, item), defences = {} }
		local i = 1
		while popup["def" .. i .. "Check"] do
			table.insert(rec.defences, { label = popup["def" .. i .. "Label"].label:gsub("^%^7", ""), min = popup["def" .. i .. "Min"].buf })
			i = i + 1
		end
		rec.default = search()
		-- every box set: Listed "In Person (Online)", a league to encode, the base, item levels and each defence row
		popup.leagueDrop:SetList({ "Rise of the Abyssal" })
		popup.listedDrop:SetSel(3)
		if not isUnique then
			popup.baseTypeCheck.state = true
			popup.ilvlMin:SetText("70")
			popup.ilvlMax:SetText("85")
			for j = 1, i - 1 do
				popup["def" .. j .. "Check"].state = true
				if j == 1 then
					popup["def" .. j .. "Max"]:SetText("5000")
				end
			end
		end
		rec.set = search()
		-- every searchable mod row checked, the odd rows with a Max, over the popup as it opens
		popup.leagueDrop:SetList({ "Standard" })
		popup.listedDrop:SetSel(1)
		if not isUnique then
			popup.baseTypeCheck.state = false
			popup.ilvlMin:SetText("")
			popup.ilvlMax:SetText("")
			for j = 1, i - 1 do
				popup["def" .. j .. "Check"].state = false
			end
		end
		rec.mods = {}
		local m = 1
		while popup["mod" .. m .. "Check"] do
			local entry = buySimilar.lastModEntries[m]
			local check = popup["mod" .. m .. "Check"]
			local row = { label = popup["mod" .. m .. "Label"].label, type = entry.type, isOption = entry.isOption, invert = entry.invert,
				tradeIds = entry.tradeIds, value = entry.value, canSearch = check:IsEnabled() }
			if popup["mod" .. m .. "Min"] then
				row.min = popup["mod" .. m .. "Min"].buf
			end
			if row.canSearch then
				check.state = true
				if m % 2 == 1 and popup["mod" .. m .. "Max"] then
					popup["mod" .. m .. "Max"]:SetText("999")
				end
			end
			table.insert(rec.mods, row)
			m = m + 1
		end
		rec.modSearch = search()
		out[slotName] = rec
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
