local outPath = arg[1]
local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local itemsTab = build.itemsTab
itemsTab.showStatDifferences = false
local result = {}
for _, db in ipairs({ main.uniqueDB, main.rareDB }) do
	for name, item in pairs(db.list) do
		local tooltip = construct("Tooltip")
		itemsTab:AddItemTooltip(tooltip, item, nil, true)
		local lines = {}
		for _, l in ipairs(tooltip.lines) do
			lines[#lines + 1] = l.text or "--"
		end
		result[name] = lines
	end
end
local keys = {}
for k in pairs(result) do
	keys[#keys + 1] = k
end
table.sort(keys)
local f = assert(io.open(outPath, "w"))
f:write(json.encode(result, { indent = true, keyorder = keys }))
f:close()
