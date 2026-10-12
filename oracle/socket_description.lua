dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local cases = { "R", "R-G-B", "R-G-B B", "R G B", "W-W A", "R-R-R-R-R-R", "B-G R-W", "A A", "G-B-R R-R B" }
local out = { }
for _, sockets in ipairs(cases) do
	local item = new("Item", "Rarity: RARE\nTest Item\nVaal Regalia\nSockets: " .. sockets .. "\nImplicits: 0")
	out[sockets] = build.itemsTab:GetSocketDescriptionLine(item)
end
print(json.encode(out))
