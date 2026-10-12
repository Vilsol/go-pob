local outPath = arg[1]
if not pcall(require, "zlib") then package.preload.zlib = function() return {} end end
if not pcall(require, "lua-utf8") then package.preload["lua-utf8"] = function() return string end end
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local lines = {
	"Increases and Reductions to Effect of Flasks applied to you also applies to Effect of Arcane Surge on you at 200% of their value",
	"On non-channelling Attack, set a Life Flask with greater than 50% of maximum Charges remaining to 50% For each Charge removed this way, that Attack gains +2% to Damage over time Multiplier",
	"Inflict Fire Exposure on Hit against Enemies with 5 Cinderflame, applying -25% to Fire Resistance",
	"When you Kill an Enemy Cursed with a Non-Aura Hex, become Immune to Curses for remaining Hex Duration",
	"When you Kill an Enemy Affected by a Non-Aura Hex, become Immune to Curses for remaining Hex Duration",
	"Non-Aura Hexes expire upon reaching 200% of base Effect Non-Aura Hexes gain 20% increased Effect per second",
	"Inflict 2 additional Poisons on the same Target when you inflict Poison with this weapon",
	"30% chance to Trigger Socketed Spells when you Spend at least 100 Mana on an Upfront Cost to Use or Trigger a Skill, with a 0.1 second Cooldown",
	"50% chance to Cast Socketed Spells when you Spend at least 100 Life on an Upfront Cost to Use or Trigger a Skill, with a 0.1 second Cooldown",
	"Deal no Damage when not on Low Life",
	"Regenerate 2% of Life per second for each different Ailment affecting you",
	"Hits have 35% chance to ignore Enemy Physical Damage Reduction while you have Sacrificial Zeal",
}
local result = {}
for _, l in ipairs(lines) do
	local list, extra = modLib.parseMod(l)
	local mods = {}
	for _, m in ipairs(list or {}) do
		local tags = {}
		for _, t in ipairs(m) do
			tags[#tags + 1] = t.type
		end
		local value = m.value
		if type(value) == "table" then
			value = value.mod and value.mod.value or 0
		elseif type(value) == "boolean" then
			value = value and 1 or 0
		end
		mods[#mods + 1] = { name = m.name, type = m.type, value = value, tags = tags }
	end
	result[#result + 1] = { line = l, extra = extra or "", mods = mods }
end
local out = assert(io.open(outPath, "w"))
out:write(json.encode(result, { indent = true }))
out:close()
print("wrote " .. outPath)
