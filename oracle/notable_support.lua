local outPath = arg[1]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local tree = new("PassiveTree", latestTreeVersion)
local out = {}
for id, node in pairs(tree.nodes) do
	if node.recipe and #node.recipe >= 1 then
		local lines, flags = {}, {}
		for i, line in ipairs(node.sd) do
			lines[i] = line
			flags[i] = (line ~= " " and (node.mods[i].extra or not node.mods[i].list)) and true or false
		end
		out[tostring(id)] = { name = node.dn, sd = lines, unsupported = flags }
	end
end
io.open(outPath, "w"):write(json.encode(out))
