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

local result = {}
for _, path in ipairs(files) do
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	local name = path:match("([^/]+)$")
	wipeGlobalCache()
	loadBuildFromXML(xmlText, name)
	runCallback("OnFrame")
	assert(build.buildName == name, "the pin did not load " .. name)
	build.itemsTab:UpdateSockets()
	local list = build.itemsTab.controls.itemList
	list:UpdateLoadoutList()
	local filters = {}
	for k, v in ipairs(list.controls.loadoutFilter.list) do
		filters[k] = v
	end
	local lists = {}
	for k, v in ipairs(filters) do
		list.controls.loadoutFilter.selIndex = k
		list:UpdateList()
		local ids = {}
		for i, id in ipairs(list.list) do
			ids[i] = id
		end
		lists[k] = { filter = v, items = ids }
	end
	result[name] = { filters = filters, lists = lists }
end
local keys = {}
for k in pairs(result) do
	keys[#keys + 1] = k
end
table.sort(keys)
local f = assert(io.open(outPath, "w"))
f:write(json.encode(result, { indent = true, keyorder = keys }))
f:close()
