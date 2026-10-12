local selfPath = arg[0]
local outPath = arg[1]
assert(outPath, "usage: gem_tooltip.lua <out.json>")
local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local gemTooltip = LoadModule("Classes/GemTooltip")

local function newTooltip()
	local lines = { }
	local tooltip = { lines = lines }
	tooltip.CheckForUpdate = function() return true end
	tooltip.AddLine = function(_, _, text)
		for line in (text .. "\n"):gmatch("([^\n]*)\n") do
			lines[#lines + 1] = { text = line }
		end
	end
	tooltip.AddSeparator = function()
		if lines[#lines] and lines[#lines].separator then
			return
		end
		lines[#lines + 1] = { separator = true }
	end
	return tooltip
end

local ids = { }
for id in pairs(data.gems) do
	ids[#ids + 1] = id
end
table.sort(ids)

local tab = build.skillsTab
build.calcsTab.GetMiscCalculator = function() return nil end
local out = { }
for _, id in ipairs(ids) do
	local gemData = data.gems[id]
	local levels = { 1 }
	if (gemData.naturalMaxLevel or 1) > 1 then
		levels[2] = gemData.naturalMaxLevel
	end
	for _, level in ipairs(levels) do
		for _, quality in ipairs({ 0, 20 }) do
			local tooltip = newTooltip()
			gemTooltip.AddGemTooltip(tooltip, build, {
				level = level, quality = quality, qualityId = "Default", count = 1, enabled = true,
				enableGlobal1 = true, enableGlobal2 = true, gemId = id, nameSpec = gemData.name,
				skillId = gemData.grantedEffectId, variantId = gemData.variantId, gemData = gemData,
			})
			out[#out + 1] = { gemId = gemData.gameId or id, skillId = gemData.grantedEffectId, variantId = gemData.variantId, level = level, quality = quality, lines = tooltip.lines }
		end
	end

	local group = { label = "", enabled = true, gemList = { {
		nameSpec = gemData.name, gemId = id, skillId = gemData.grantedEffectId, variantId = gemData.variantId, level = 1, quality = 0, qualityId = "Default",
		enabled = true, enableGlobal1 = true, enableGlobal2 = true, count = 1,
	} } }
	tab:ProcessSocketGroup(group)
	tab:SetDisplayGroup(group)
	local tooltip = newTooltip()
	tab.controls.gemSlot1Quality.tooltipFunc(tooltip)
	out[#out + 1] = { gemId = gemData.gameId or id, skillId = gemData.grantedEffectId, variantId = gemData.variantId, qualityTooltip = true, lines = tooltip.lines }
end
assert(#out > 1000, "REFUSING: only " .. #out .. " tooltips")

local file = assert(io.open(outPath, "w"))
file:write(json.encode(stable({
	characterLevel = build.characterLevel,
	str = build.calcsTab.mainOutput.Str, dex = build.calcsTab.mainOutput.Dex, int = build.calcsTab.mainOutput.Int,
	showFlavourText = main.showFlavourText,
	tooltips = out,
})))
file:close()
io.stderr:write(string.format("wrote %s (%d tooltips)\n", outPath, #out))
