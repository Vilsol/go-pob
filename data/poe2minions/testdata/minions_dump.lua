-- Prints PoB-PoE2's loaded data.minions as `key<TAB>field<TAB>value` rows, read straight off the tables
-- Data.lua builds, for the Go-side differential in minions_pob_test.go.
dofile("HeadlessWrapper.lua")

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys, function(a, b)
		if type(a) == type(b) then
			return a < b
		end
		return type(a) == "number"
	end)
	return keys
end

local function canon(v)
	if type(v) == "number" then
		return string.format("%.14g", v)
	elseif type(v) == "string" then
		return '"' .. v .. '"'
	elseif type(v) == "boolean" then
		return tostring(v)
	elseif type(v) == "table" then
		local parts = {}
		for _, k in ipairs(sortedKeys(v)) do
			parts[#parts + 1] = tostring(k) .. "=" .. canon(v[k])
		end
		return "{" .. table.concat(parts, ",") .. "}"
	end
	return "nil"
end

for _, key in ipairs(sortedKeys(data.minions)) do
	local m = data.minions[key]
	local function out(field, v)
		io.write(key, "\t", field, "\t", v, "\n")
	end
	out("name", m.name)
	out("limit", m.limit or "")
	out("extraFlags", canon(m.extraFlags or {}))
	out("skillList", canon(m.skillList))
	out("accuracy", canon(m.accuracy))
	for i, mod in ipairs(m.modList) do
		out("mod" .. i, canon(mod))
	end
	out("mods", tostring(#m.modList))
end
