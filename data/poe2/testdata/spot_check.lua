-- Prints PoB-PoE2's loaded data tables as `kind<TAB>key<TAB>field<TAB>value` rows, read straight off
-- the tables Data.lua builds, for the Go-side differential in pob_spot_check_test.go.
dofile("HeadlessWrapper.lua")

local function out(kind, key, field, v)
	if v == nil then
		return
	end
	if type(v) == "number" then
		v = string.format("%.17g", v)
	elseif type(v) == "boolean" then
		v = tostring(v)
	end
	io.write(kind, "\t", key, "\t", field, "\t", (v:gsub("\n", "\\n")), "\n")
end

local n = 0
for name, b in pairs(data.itemBases) do
	n = n + 1
	out("base", name, "type", b.type)
	out("base", name, "subType", b.subType)
	out("base", name, "implicit", b.implicit)
	out("base", name, "quality", b.quality)
	out("base", name, "socketLimit", b.socketLimit)
	out("base", name, "req.level", b.req.level)
	out("base", name, "req.str", b.req.str)
	out("base", name, "req.dex", b.req.dex)
	out("base", name, "req.int", b.req.int)
	for _, f in ipairs({ "Armour", "Evasion", "EnergyShield", "Ward", "BlockChance", "MovementPenalty" }) do
		out("base", name, "armour." .. f, b.armour and b.armour[f])
	end
	for _, f in ipairs({ "PhysicalMin", "PhysicalMax", "CritChanceBase", "AttackRateBase", "Range", "ReloadTimeBase" }) do
		out("base", name, "weapon." .. f, b.weapon and b.weapon[f])
	end
end
out("count", "base", "rows", n)

n = 0
for name, slots in pairs(data.itemMods.Runes) do
	for slot, mod in pairs(slots) do
		n = n + 1
		local key = name .. "/" .. slot
		out("rune", key, "type", mod.type)
		out("rune", key, "lines", table.concat(mod, "|"))
		out("rune", key, "levelReq", mod.levelReq)
		out("rune", key, "limit", mod.limit)
	end
end
out("count", "rune", "rows", n)

n = 0
for _, t in ipairs({ "misc", "gameConstants", "characterConstants", "monsterConstants" }) do
	for k, v in pairs(data[t]) do
		n = n + 1
		out("const", t, k, v)
	end
end
for i, v in ipairs(data.monsterLifeTable) do
	n = n + 1
	out("const", "monsterLifeTable", tostring(i), v)
end
out("count", "const", "rows", n)

n = 0
for id, gem in pairs(data.gems) do
	n = n + 1
	out("gem", id, "name", gem.name)
	out("gem", id, "grantedEffectId", gem.grantedEffectId)
	out("gem", id, "support", gem.grantedEffect.support == true)
	out("gem", id, "tagString", gem.tagString)
	out("gem", id, "reqStr", gem.reqStr)
	out("gem", id, "reqDex", gem.reqDex)
	out("gem", id, "reqInt", gem.reqInt)
	out("gem", id, "naturalMaxLevel", gem.naturalMaxLevel)
end
out("count", "gem", "rows", n)
