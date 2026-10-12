local selfPath = arg[0]
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: custom_mod_lines.lua <cases.txt> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local lines = {}
local f = assert(io.open(casesPath, "r"))
for line in f:lines() do
	table.insert(lines, line)
end
f:close()

local results = {}
for i, line in ipairs(lines) do
	local strippedLine = StripEscapes(line):match("^%s*(.-)%s*$")
	modLib.parseModCache[strippedLine] = nil
	local mods, extra = modLib.parseMod(strippedLine)
	local rec = { line = line, parsed = (mods and not extra) and true or false, extra = extra or "", mods = {} }
	if mods and not extra then
		for j, mod in ipairs(mods) do
			local tags = {}
			for _, tag in ipairs(mod) do
				table.insert(tags, tag.type)
			end
			rec.mods[j] = {
				name = mod.name,
				type = mod.type,
				value = type(mod.value) ~= "table" and tostring(mod.value) or "",
				tags = tags,
			}
		end
	end
	results[i] = rec
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()

io.stderr:write(string.format("custom_mod_lines: %d lines -> %s\n", #results, outPath))
