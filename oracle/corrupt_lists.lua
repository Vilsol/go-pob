local inPath, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local items = json.decode(io.open(inPath):read("*a"))
local out = {}
for k, raw in ipairs(items) do
	local item = new("Item", raw)
	local res = {}
	for _, modType in ipairs({ "Corrupted", "ScourgeUpside", "ScourgeDownside" }) do
		local list = {}
		for modId, mod in pairs(item.affixes) do
			if mod.type == modType and item:GetModSpawnWeight(mod) > 0 then
				table.insert(list, { id = modId, mod = mod })
			end
		end
		table.sort(list, function(a, b)
			local an = a.mod[1]:lower():gsub("%(.-%)","$"):gsub("[%+%-%%]",""):gsub("%d+","$")
			local bn = b.mod[1]:lower():gsub("%(.-%)","$"):gsub("[%+%-%%]",""):gsub("%d+","$")
			if an ~= bn then return an < bn else return a.mod.level < b.mod.level end
		end)
		local ids = {}
		for i, e in ipairs(list) do ids[i] = e.id end
		res[modType] = ids
	end
	out[k] = res
end
io.open(outPath, "w"):write(json.encode(out))
