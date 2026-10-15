local compat = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua")
dofile("HeadlessWrapper.lua")
newBuild()

local displayStats = compat.loadBuildDisplayStats()

local fmts, seen = { }, { }
for _, entry in ipairs(displayStats) do
	if entry.fmt and not seen[entry.fmt] then
		seen[entry.fmt] = true
		fmts[#fmts + 1] = entry.fmt
	end
end
table.sort(fmts)

local values = { 0, 1, -1, 0.5, -0.5, 2.5, 0.125, 1.005, 12.345, 99.995, 999.95, 1234.5, -1234.5, 99999.99, 1234567.891, 123456789012, 1/3, 75.25, 100 }
local overCaps = { false, 0, 12.7 }

local function str(v)
	return '"' .. tostring(v):gsub('\\', '\\\\'):gsub('"', '\\"') .. '"'
end
local function num(v)
	return string.format("%.17g", v)
end

local out = { }
for _, fmt in ipairs(fmts) do
	for _, flags in ipairs({ { }, { pc = true }, { mod = true } }) do
		for vi, v in ipairs(values) do
			local overCap = overCaps[(vi % #overCaps) + 1]
			local statData = { fmt = fmt, pc = flags.pc, mod = flags.mod, label = "Case" }
			local result = build:FormatStat(statData, v, overCap or nil)
			out[#out + 1] = "{" .. table.concat({
				'"fmt":' .. str(fmt),
				'"pc":' .. tostring(flags.pc == true),
				'"mod":' .. tostring(flags.mod == true),
				'"label":"Case"',
				'"value":' .. num(v),
				'"overCap":' .. (overCap and num(overCap) or "null"),
				'"out":' .. str(result),
			}, ",") .. "}"
		end
	end
end
for _, v in ipairs({ 0, 5, -5 }) do
	local result = build:FormatStat({ fmt = "d", label = "Unreserved Life" }, v)
	out[#out + 1] = '{"fmt":"d","pc":false,"mod":false,"label":"Unreserved Life","value":' .. num(v) .. ',"overCap":null,"out":' .. str(result) .. '}'
end
print("[\n" .. table.concat(out, ",\n") .. "\n]")
