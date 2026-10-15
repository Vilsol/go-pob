local which = arg[1]
local compat = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua")
dofile("HeadlessWrapper.lua")

local displayStats, minionDisplayStats = compat.loadBuildDisplayStats()
if which == "minion" then
	displayStats = minionDisplayStats
end

local function str(v)
	return '"' .. tostring(v):gsub('\\', '\\\\'):gsub('"', '\\"') .. '"'
end

local keys = { "stat", "childStat", "label", "fmt", "flag", "notFlag", "color", "overCapStat", "suffix", "breakdown" }
local lists = { "modNames", "ignoredSections" }
local bools = { "pc", "mod", "hideStat", "warnColor", "chaosInoc" }
local funcs = { "condFunc", "warnFunc", "suffixCondFunc", "overCapStatCondFunc" }

local out = {}
for _, entry in ipairs(displayStats) do
	local fields = {}
	for _, k in ipairs(keys) do
		local v = entry[k]
		if type(v) == "function" then
			fields[#fields + 1] = str(k) .. ':"<function>"'
		elseif v ~= nil then
			if k == "color" then
				v = v:gsub("^%^x", "#")
			end
			fields[#fields + 1] = str(k) .. ":" .. str(v)
		end
	end
	for _, k in ipairs(lists) do
		if entry[k] then
			local items = {}
			for i, v in ipairs(entry[k]) do
				items[i] = str(v)
			end
			fields[#fields + 1] = str(k) .. ":[" .. table.concat(items, ",") .. "]"
		end
	end
	for _, k in ipairs(bools) do
		if entry[k] then
			fields[#fields + 1] = str(k) .. ":true"
		end
	end
	for _, k in ipairs(funcs) do
		if entry[k] then
			fields[#fields + 1] = str(k) .. ":true"
		end
	end
	out[#out + 1] = "{" .. table.concat(fields, ",") .. "}"
end
print("[\n" .. table.concat(out, ",\n") .. "\n]")
