local outPath = arg[1]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local isKeyDown = IsKeyDown
function IsKeyDown(key)
	return key == "SHIFT" or isKeyDown(key)
end

local itemsTab = build.itemsTab
itemsTab.showStatDifferences = false
local result = {}
for name, item in pairs(main.uniqueDB.list) do
	local tooltip = new("Tooltip")
	itemsTab:AddItemTooltip(tooltip, item, nil, true)
	local children = {}
	for _, child in ipairs(tooltip.childTooltips or {}) do
		if #child.lines > 0 then
			local lines = {}
			for _, l in ipairs(child.lines) do
				lines[#lines + 1] = l.text or "--"
			end
			children[#children + 1] = lines
		end
	end
	if #children > 0 then
		result[name] = children
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
