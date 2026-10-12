
dofile("HeadlessWrapper.lua")

local RAW = [[
Rarity: UNIQUE
Atziri's Step
Slink Boots
LevelReq: 69
Implicits: 0
+(55-75) to maximum Life
]]

local RAW_TAGGED = [[
Rarity: UNIQUE
Atziri's Step
Slink Boots
LevelReq: 69
Implicits: 0
{range:0.25}+(55-75) to maximum Life
]]

local function lifeValue(item)
	item:BuildModList()
	for _, m in ipairs(item.baseModList or {}) do
		if m.name == "Life" and m.type == "BASE" then
			return m.value
		end
	end
	return nil
end

local function probe(label, raw)
	local item = new("Item", raw)
	local line = item.explicitModLines[1]
	print(string.format("  %-34s range=%-8s Life BASE=%s",
		label, tostring(line and line.range), tostring(lifeValue(item))))
end

print("")
print("=== pinned PoB, headless, as the oracle dumper runs it ===")
print(string.format("  main.defaultItemAffixQuality = %s   (type %s)",
	tostring(main.defaultItemAffixQuality), type(main.defaultItemAffixQuality)))
print(string.format("  main.userPath                = %s", tostring(main.userPath)))
local f = io.open((main.userPath or "") .. "Settings.xml", "r")
print(string.format("  Settings.xml present         = %s", tostring(f ~= nil)))
if f then f:close() end

print("")
print("=== +(55-75) to maximum Life, no {range:} tag ===")
print("  (55-75): midpoint would be 65, maximum 75)")
probe("as-loaded (setting untouched)", RAW)

main.defaultItemAffixQuality = 0.5
probe("forced defaultItemAffixQuality=0.5", RAW)

main.defaultItemAffixQuality = 1
probe("forced defaultItemAffixQuality=1", RAW)

main.defaultItemAffixQuality = nil
probe("forced defaultItemAffixQuality=nil", RAW)

print("")
print("=== the same line WITH an explicit {range:0.25} tag ===")
print("  (55 + 0.25*(75-55) = 60)")
probe("as-loaded (setting untouched)", RAW_TAGGED)
main.defaultItemAffixQuality = 0.5
probe("forced defaultItemAffixQuality=0.5", RAW_TAGGED)
print("")
