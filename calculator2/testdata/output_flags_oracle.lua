-- Loads each build XML and prints one TSV row per boolean or string entry of the main calculation's
-- player output: build, key, type, value.
-- usage: output_flags_oracle.lua <out.tsv> <xml>...
local outPath = arg[1]
assert(outPath and arg[2], "usage: output_flags_oracle.lua <out.tsv> <xml>...")
local inputs = {}
while arg[#inputs + 2] do
	inputs[#inputs + 1] = arg[#inputs + 2]
end

dofile("HeadlessWrapper.lua")

local out = assert(io.open(outPath, "w"))
for _, input in ipairs(inputs) do
	local f = assert(io.open(input, "r"))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml)
	runCallback("OnFrame")
	local name = input:match("[^/]+$")
	out:write(name, "\tbuild\tbuild\t\n")
	for k, v in pairs(build.calcsTab.mainEnv.player.output) do
		local t = type(v)
		if t == "boolean" or t == "string" then
			out:write(name, "\t", k, "\t", t, "\t", (tostring(v):gsub("[\t\n]", " ")), "\n")
		end
	end
end
out:close()
