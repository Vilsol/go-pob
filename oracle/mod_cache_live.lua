
local outPath, extraPath = arg[1], arg[2]
assert(outPath, "usage: mod_cache_live.lua <out.lua> [extra-lines.txt]")

dofile("HeadlessWrapper.lua")

local shipped = {}
local chunk = assert(loadfile("Data/ModCache.lua"))
chunk(shipped)

local lines = {}
for line in pairs(shipped) do
	table.insert(lines, line)
end
local shippedCount = #lines
if extraPath then
	for line in io.lines(extraPath) do
		if line ~= "" and not shipped[line] then
			shipped[line] = true
			table.insert(lines, line)
		end
	end
end
table.sort(lines)

local function esc(s)
	return (s:gsub("\\", "\\\\"):gsub("\n", "\\n"):gsub('"', '\\"'))
end

local cache = modLib.parseModCache
local out = assert(io.open(outPath, "w"))
out:write('local c=...')
local written = 0
for _, line in ipairs(lines) do
	cache[line] = nil
	local modList, extra = modLib.parseMod(line)
	if not modList or not modList[1] or (modList[1].name ~= "JewelFunc" and modList[1].name ~= "ExtraJewelFunc") then
		out:write('c["', esc(line), '"]={')
		if modList then
			writeLuaTable(out, modList)
		else
			out:write('nil')
		end
		if extra then
			out:write(',"', esc(extra), '"}\n')
		else
			out:write(',nil}\n')
		end
		written = written + 1
	end
end
out:close()
print(string.format("mod_cache_live: %d shipped keys, %d extra, %d written", shippedCount, #lines - shippedCount, written))
