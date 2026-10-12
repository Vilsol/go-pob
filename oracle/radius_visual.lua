local outPath = assert(arg[1], "usage: radius_visual.lua <out.json>")
local lines = {}
for line in io.lines("Modules/Main.lua") do
	lines[#lines + 1] = line
end
assert(lines[1490]:match("function main:WorldToScreen"), "REFUSING: Main.lua:1490 is no longer WorldToScreen")
assert(lines[1502]:match("function main:RenderCircle"), "REFUSING: Main.lua:1502 is no longer RenderCircle")
local src = { "local main, DrawImage, round, wipeTable, tempTable1, tempTable2, m_pi, m_sin, m_cos, m_min, m_max = ..." }
for i = 1486, 1523 do
	src[#src + 1] = lines[i]
end
src[#src + 1] = "return main"
local rows
local main = assert(loadstring(table.concat(src, "\n")))({}, function(_, x, y, w)
	rows[#rows + 1] = { y, x, w }
end, function(v) return math.floor(v + 0.5) end, function(t)
	for k in pairs(t) do t[k] = nil end
	return t
end, {}, {}, math.pi, math.sin, math.cos, math.min, math.max)
local out = {}
for _, radius in ipairs({ 6, 8, 11, 16, 23, 30, 40, 52, 70 }) do
	rows = {}
	main:RenderCircle(0, 0, 480, 270, 0, 0, radius)
	out[#out + 1] = { radius = radius, rows = rows }
end
local json = require("dkjson")
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out))
f:close()
