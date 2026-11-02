-- usage (from PoB-PoE2's src/): luajit poe2_pages.lua <build.xml> <out.json>
-- Records the Skills and Items tabs PoB-PoE2 shows for a build: each socket group's list label, slot,
-- source, flags and gems, and the item in every shown slot with its tooltip title and mod lines.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_pages.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local groups = {}
local tab = build.skillsTab
for _, group in ipairs(tab.socketGroupList) do
	-- the gem slot controls read the displayed group
	tab:SetDisplayGroup(group)
	tab:UpdateGemSlots()
	local gems = {}
	for i, gem in ipairs(group.gemList) do
		local grantedEffect = gem.grantedEffect or (gem.gemData and gem.gemData.grantedEffect)
		local slot = tab.gemSlots[i]
		gems[#gems + 1] = {
			id = gem.gemData and gem.gemData.id or "",
			generated = not slot.nameSpec.enabled(),
			countShown = slot.count.shown() and true or false,
			corruptShown = slot.corruptLevel.shown() and true or false,
			corruptIndex = tab:GetCorruptIndex(gem),
			name = gem.gemData and gem.gemData.name or (grantedEffect and grantedEffect.name) or gem.nameSpec,
			level = gem.level,
			quality = gem.quality,
			count = gem.count or 1,
			enabled = gem.enabled and true or false,
			support = grantedEffect and grantedEffect.support and true or false,
			color = gem.color,
		}
	end
	groups[#groups + 1] = {
		label = group.displayLabel,
		userLabel = group.label or "",
		weaponSetLocked = tab:IsSocketGroupWeaponSetLocked(group) and true or false,
		groupCount = group.groupCount or 1,
		slot = group.slot or "",
		source = group.source or "",
		enabled = group.enabled and true or false,
		includeInFullDPS = group.includeInFullDPS and true or false,
		gems = gems,
	}
end

-- ItemsTab:AddItemTooltip into a tooltip that keeps lines and separators
local items = {}
local itemsTab = build.itemsTab
for _, slot in ipairs(itemsTab.orderedSlots) do
	local item = itemsTab.items[slot.selItemId]
	if slot.selItemId ~= 0 and item and slot:IsShown() then
		local lines, group = {}, {}
		local tip = {
			AddLine = function(_, _, text) group[#group + 1] = text end,
			AddSeparator = function()
				if #group > 0 then
					lines[#lines + 1] = group
					group = {}
				end
			end,
			CheckForUpdate = function() return true end,
			Clear = function() lines, group = {}, {} end,
		}
		itemsTab:AddItemTooltip(tip, item, slot)
		if #group > 0 then
			lines[#lines + 1] = group
		end
		-- the title and mod line groups AddItemTooltip draws (ItemsTab.lua:4039-4141 without a jewel radius
		-- scale), each checked against the tooltip it drew
		local mods = {}
		for _, modList in ipairs({ item.enchantModLines, item.runeModLines, item.implicitModLines, item.explicitModLines }) do
			if modList[1] then
				local g = {}
				for _, modLine in ipairs(modList) do
					local formatted = item:GetModLineVariantCount(modLine) > 0 and itemLib.formatModLine(modLine)
					if formatted then
						for _ = 1, item:GetModLineVariantCount(modLine) do
							g[#g + 1] = formatted
						end
					end
				end
				mods[#mods + 1] = g
			end
		end
		local labels = {}
		if item.mirrored then labels[#labels + 1] = colorCodes.NEGATIVE .. "Mirrored" end
		if item.sanctified then labels[#labels + 1] = colorCodes.FRACTURED .. "Sanctified" end
		if item.doubleCorrupted then
			labels[#labels + 1] = colorCodes.NEGATIVE .. "Twice Corrupted"
		elseif item.corrupted then
			labels[#labels + 1] = colorCodes.NEGATIVE .. "Corrupted"
		end
		if #labels > 0 then mods[#mods + 1] = labels end
		local drawn = {}
		for _, g in ipairs(lines) do drawn[table.concat(g, "\n")] = true end
		for _, g in ipairs(mods) do
			assert(#g == 0 or drawn[table.concat(g, "\n")], "item " .. item.id .. ": a mod line group the tooltip did not draw: " .. table.concat(g, " | "))
		end
		items[#items + 1] = { slot = slot.slotName, id = item.id, rarity = item.rarity, name = lines[1], lines = mods }
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ groups = groups, useSecondWeaponSet = itemsTab.activeItemSet.useSecondWeaponSet and true or false, items = items }), { indent = true }))
o:write("\n")
o:close()
