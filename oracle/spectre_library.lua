local outPath = arg[1]
assert(outPath, "usage: spectre_library.lua <out.json>")
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local data = build.data
local sourceList = { }
for id in pairs(data.spectres) do
	table.insert(sourceList, id)
end
table.sort(sourceList, function(a, b)
	if data.minions[a].name == data.minions[b].name then
		return a < b
	else
		return data.minions[a].name < data.minions[b].name
	end
end)
local list = construct("MinionListControl", nil, {0, 0, 10, 10}, data, { })
local out = { }
for _, id in ipairs(sourceList) do
	local lines = { }
	local tooltip = { CheckForUpdate = function() return true end, AddLine = function(_, _, text) lines[#lines + 1] = text end }
	list:AddValueTooltip(tooltip, 1, id)
	out[#out + 1] = { id = id, lines = lines }
end
assert(#out > 0, "REFUSING: data.spectres is empty")
local file = assert(io.open(outPath, "w"))
file:write(json.encode(out))
file:close()
io.stderr:write(string.format("wrote %s (%d spectres)\n", outPath, #out))
