
local buildPath = arg[1]
assert(buildPath, "usage: magic_affix_reroll.lua <buildXML>")

dofile("HeadlessWrapper.lua")

local RAW = [[
Rarity: MAGIC
Vivid Cobalt Jewel of the Phoenix
Unique ID: 441b7094f5c4943ec58d89f90f1536cb598728adbcc13e7322bff44e98fbf40b
Item Level: 82
Implicits: 0
7% increased maximum Life
+2% to maximum Fire Resistance
Corrupted
]]

local function report(label, item)
	item:BuildModList()
	print("  " .. label)
	for i, line in ipairs(item.explicitModLines or {}) do
		print(string.format("    line[%d] range=%-6s valueScalar=%-6s  %s",
			i, tostring(line.range), tostring(line.valueScalar), tostring(line.line)))
	end
	for _, m in ipairs(item.baseModList or {}) do
		if m.name == "FireResistMax" or m.name == "Life" then
			print(string.format("    mod  %-16s %-8s %s", m.name, m.type, tostring(m.value)))
		end
	end
end

print("")
print("=== A. standalone parse of the exact text ===")
report("standalone", new("Item", RAW))

local fh = assert(io.open(buildPath, "r"))
local xmlText = fh:read("*a")
fh:close()
loadBuildFromXML(xmlText, "probe")
runCallback("OnFrame")

print("")
print("=== B. the same item inside the loaded build ===")
local item = build.itemsTab.items[10]
if not item then
	print("  REFUSING: no item id 10 in the loaded build")
else
	print(string.format("  name=%s rarity=%s", tostring(item.name), tostring(item.rarity)))
	report("in-build", item)
end
print("")
