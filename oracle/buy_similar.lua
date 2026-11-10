-- usage (from PoB's src/): luajit buy_similar.lua <out.json> <build.xml>...
-- Records CompareBuySimilar.openPopup's popup for every item each build equips, as the Compare tab opens it for a
-- compared item: whether it is unique, its category label, base name and defence rows, and the trade URL the Open URL
-- button copies with the popup as it opens, with every box set, and with every searchable mod row checked, its query
-- decoded, and its mod rows.
local selfPath, outPath = arg[0], arg[1]
assert(outPath and arg[2], "usage: buy_similar.lua <out.json> <build.xml>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local buySimilar = require("Classes.CompareBuySimilar")
-- the popup's mod entries, which it keeps local
local addModEntries = buySimilar.addModEntries
buySimilar.addModEntries = function(...)
	buySimilar.lastModEntries = addModEntries(...)
	return buySimilar.lastModEntries
end
local tradeHelpers = require("Classes.TradeHelpers")

-- the league list is not fetched: TradeQueryRequests:FetchLeagues falls back to Standard
main.api = main.api or {}
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

local function search()
	-- a control's callback rebuilds the URL, as the league list arriving does
	popup.listedDrop.selFunc(popup.listedDrop.selIndex, popup.listedDrop.list[popup.listedDrop.selIndex])
	copied = nil
	popup.search.onClick()
	local base, encoded = copied:match("^(.-)%?q=(.*)$")
	return { base = base, query = json.decode((urlDecode(encoded))) }
end

local out = {}
for i = 2, #arg do
	local path = arg[i]
	local f = assert(io.open(path))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml)
	runCallback("OnFrame")
	local recs = {}
	for slotName, slot in pairs(build.itemsTab.slots) do
		local item = build.itemsTab.items[slot.selItemId]
		if item then
			-- the popup remembers the last realm, Listed and league choice; each item opens as the first does
			buySimilar.lastListedIndex, buySimilar.lastRealmIdx, buySimilar.lastLeagueByRealm = nil, nil, nil
			buySimilar.openPopup(item, slotName, build)
			local isUnique = item.rarity == "UNIQUE" or item.rarity == "RELIC"
			local rec = { unique = isUnique, name = item.name, baseName = item.baseName, category = tradeHelpers.getTradeCategoryLabel(slotName, item), defences = {} }
			local d = 1
			while popup["def" .. d .. "Check"] do
				table.insert(rec.defences, { label = popup["def" .. d .. "Label"].label:gsub("^%^7", ""), min = popup["def" .. d .. "Min"].buf })
				d = d + 1
			end
			rec.default = search()
			-- every box set: realm Xbox, Listed "In Person (Online)", a league to encode, the base, item levels and each
			-- defence row
			popup.realmDrop:SetSel(3)
			popup.leagueDrop:SetList({ "Mirage Ruthless" })
			popup.listedDrop:SetSel(3)
			if not isUnique then
				popup.baseTypeCheck.state = true
				popup.ilvlMin:SetText("70")
				popup.ilvlMax:SetText("85")
				for j = 1, d - 1 do
					popup["def" .. j .. "Check"].state = true
					if j == 1 then
						popup["def" .. j .. "Max"]:SetText("5000")
					end
				end
			end
			rec.set = search()
			-- every searchable mod row checked, the odd rows with a Max, over the popup as it opens
			popup.realmDrop:SetSel(1)
			popup.leagueDrop:SetList({ "Standard" })
			popup.listedDrop:SetSel(1)
			if not isUnique then
				popup.baseTypeCheck.state = false
				popup.ilvlMin:SetText("")
				popup.ilvlMax:SetText("")
				for j = 1, d - 1 do
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
			recs[slotName] = rec
		end
	end
	out[path:match("([^/\\]+)$")] = recs
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:write("\n")
o:close()
