
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: mod_scalability_cases.lua <cases.json> <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"), "REFUSING: cannot read " .. casesPath)
local cases = assert(json.decode(f:read("*a")), "REFUSING: " .. casesPath .. " is not JSON")
f:close()

assert(#cases > 0, "REFUSING: the case file is empty; a probe with no cases records a golden that " ..
	"asserts nothing and passes forever")

assert(data.modScalability, "REFUSING: the pinned PoB does not expose data.modScalability, so " ..
	"every case would record the legacy fallback arm instead")

local results = {}
local scalarCases = 0
for i, c in ipairs(cases) do
	assert(type(c.line) == "string", "REFUSING: case " .. i .. " has no string `line`")
	assert(type(c.range) == "number", "REFUSING: case " .. i .. " has no numeric `range`")
	if c.valueScalar or c.baseValueScalar then
		scalarCases = scalarCases + 1
	end
	results[i] = itemLib.applyRange(c.line, c.range, c.valueScalar, c.baseValueScalar)
end

local file = assert(io.open(outPath, "w"), "REFUSING: cannot write " .. outPath)
file:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	results = results,
}, { indent = true, keyorder = { "pobVersion", "results" } }))
file:close()

io.stderr:write(string.format("wrote %s (PoB %s, %d cases, %d with scalars)\n",
	outPath, tostring(launch.versionNumber or "unknown"), #results, scalarCases))
