
local selfPath = arg[0]
local outPath = arg[1]
local files = {}
for i = 2, #arg do
	files[#files + 1] = arg[i]
end
assert(outPath and #files > 0, "usage: item_tooltip.lua <out.json> <build.xml>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

if not pcall(require, "zlib") then
	package.preload.zlib = function()
		local refuse = function() error("lua-zlib is not installed (mise run bootstrap in the PoB checkout)") end
		return { deflate = refuse, inflate = refuse }
	end
end

if not pcall(require, "lua-utf8") then
	package.preload["lua-utf8"] = function() return string end
end

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local results = {}
for _, path in ipairs(files) do
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	local name = path:match("([^/]+)$")

	local entry = { build = name }
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(xmlText, name)
		runCallback("OnFrame")
		assert(build.buildName == name, "the pin did not load it (still showing " .. tostring(build.buildName) .. ")")
		local xmlItems = 0
		for _ in xmlText:gmatch("<Item id=") do xmlItems = xmlItems + 1 end
		assert(#build.itemsTab.itemOrderList == xmlItems, "the pin loaded " .. #build.itemsTab.itemOrderList .. " items of the XML's " .. xmlItems)
		local plainXML = xmlText:gsub("&apos;", "'"):gsub("&quot;", '"'):gsub("&lt;", "<"):gsub("&gt;", ">"):gsub("&amp;", "&")
		for _, id in ipairs(build.itemsTab.itemOrderList) do
			local item = build.itemsTab.items[id]
			assert(plainXML:find(item.baseName, 1, true), "the pin kept another build's items loaded (" .. item.baseName .. " is not in the XML)")
		end
		local itemsTab = build.itemsTab
		itemsTab.showStatDifferences = false
		local env = build.calcsTab.mainEnv
		local output = build.calcsTab.mainOutput
		entry.level = build.characterLevel
		entry.className = build.spec.curClassName
		entry.stats = {
			Str = output.Str or 0,
			Dex = output.Dex or 0,
			Int = output.Int or 0,
			Omni = output.Omni or 0,
			OmniscienceRequirements = env.modDB:Flag(nil, "OmniscienceRequirements") and true or false,
			OmniAttributeRequirements = env.modDB:Sum("INC", nil, "OmniAttributeRequirements"),
		}
		local items = {}
		for _, id in ipairs(itemsTab.itemOrderList or {}) do
			local item = itemsTab.items[id]
			if item then
				local tooltip = new("Tooltip")
				itemsTab:AddItemTooltip(tooltip, item)
				local lines = {}
				for _, l in ipairs(tooltip.lines) do
					if l.text then
						lines[#lines + 1] = { text = l.text, font = l.font or "" }
					else
						lines[#lines + 1] = { sep = true }
					end
				end
				local entryItem = { id = id, rarity = item.rarity, lines = lines }
				if item.base and item.base.flask then
					itemsTab.showStatDifferences = true
					local full = new("Tooltip")
					itemsTab:AddItemTooltip(full, item)
					itemsTab.showStatDifferences = false
					local effective, on = {}, false
					for _, l in ipairs(full.lines) do
						if l.text == "^7Effective flask stats:" then
							on = true
						elseif l.text and l.text:find("this flask will give you:") then
							on = false
						end
						if on and l.text then
							effective[#effective + 1] = l.text
						end
					end
					entryItem.effective = effective
				end
				items[#items + 1] = entryItem
			end
		end
		entry.items = items
	end)
	if not ok then
		entry.error = tostring(err)
	end
	assert(entry.items or entry.error, "REFUSING: " .. name .. " produced neither items nor an error")
	results[#results + 1] = entry
	io.stderr:write(string.format("item-tooltip: %s%s\n", name, entry.error and ("  ERROR " .. entry.error) or ""))
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable({ builds = results }), { indent = true }))
out:close()
io.stderr:write("item-tooltip: wrote " .. #results .. " builds to " .. outPath .. "\n")
