local selfPath = arg[0]
local listPath, outPath = arg[1], arg[2]
assert(listPath and outPath, "usage: spec_node_rewrites.lua <list> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function sameLines(a, b)
	if #a ~= #b then
		return false
	end
	for i = 1, #a do
		if a[i] ~= b[i] then
			return false
		end
	end
	return true
end

local out = {}
for path in io.lines(listPath) do
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	local entry = {}
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(text, path:match("[^/]+$"))
		runCallback("OnFrame")
		local spec = build.spec
		for id, node in pairs(spec.nodes) do
			local orig = spec.tree.nodes[id]
			if orig and node ~= orig and (node.type ~= "Mastery" or spec.hashOverrides[id]) and (node.dn ~= orig.dn or not sameLines(node.sd or {}, orig.sd or {})) then
				local sd = {}
				for i, line in ipairs(node.sd or {}) do
					sd[i] = line
				end
				entry[tostring(id)] = { name = node.dn, stats = sd }
			end
		end
	end)
	if not ok then
		entry = { error = tostring(err) }
	end
	out[path:match("[^/]+$")] = entry
end

local f = assert(io.open(outPath, "w"))
f:write(json.encode(stable(out), { indent = true }))
f:close()
io.stderr:write("spec_node_rewrites: wrote " .. outPath .. "\n")
