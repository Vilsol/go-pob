local outPath = arg[1]
local files = {}
for i = 2, #arg do
	files[#files + 1] = arg[i]
end
if not pcall(require, "zlib") then
	package.preload.zlib = function()
		local refuse = function() error("lua-zlib is not installed (mise run bootstrap in the PoB checkout)") end
		return { deflate = refuse, inflate = refuse }
	end
end
if not pcall(require, "lua-utf8") then
	package.preload["lua-utf8"] = function() return string end
end

dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local function load(xmlText, name)
	wipeGlobalCache()
	loadBuildFromXML(xmlText, name)
	runCallback("OnFrame")
	assert(build.buildName == name, "the pin did not load " .. name)
end
local function allocated()
	local ids = {}
	for id in pairs(build.spec.allocNodes) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	return ids
end

local result = {}
for _, path in ipairs(files) do
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	local name = path:match("([^/]+)$")
	local ok, err = pcall(load, xmlText, name)
	assert(ok, "REFUSING: " .. name .. ": " .. tostring(err))
	local socketed = {}
	for nodeId, itemId in pairs(build.spec.jewels) do
		if itemId ~= 0 and build.spec.allocNodes[nodeId] and build.itemsTab.items[itemId] then
			socketed[#socketed + 1] = itemId
		end
	end
	table.sort(socketed)
	local entry = { before = allocated(), after = {} }
	for _, itemId in ipairs(socketed) do
		load(xmlText, name)
		build.itemsTab:DeleteItem(build.itemsTab.items[itemId])
		entry.after[tostring(itemId)] = allocated()
	end
	result[name] = entry
	io.stderr:write(string.format("item-delete-tree: %s, %d jewels\n", name, #socketed))
end
local keys = {}
for k in pairs(result) do
	keys[#keys + 1] = k
end
table.sort(keys)
local f = assert(io.open(outPath, "w"))
f:write(json.encode(result, { indent = true, keyorder = keys }))
f:close()
