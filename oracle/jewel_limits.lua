
local buildsDir = arg[1]
assert(buildsDir, "usage: jewel_limits.lua <buildsDir>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local files = {}
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		files[#files + 1] = file
	end
end
table.sort(files)

for _, file in ipairs(files) do
	local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		local env = build.calcsTab.mainEnv
		local sockets = {}
		local disabled = 0

		for _, slot in ipairs(build.itemsTab.orderedSlots) do
			if slot.nodeId then
				local item = build.itemsTab.items[slot.selItemId]
				if item and env.allocNodes[slot.nodeId] then
					local limitDisabled = item.jewelData and item.jewelData.limitDisabled or false
					if limitDisabled then
						disabled = disabled + 1
					end
					local stamped = 0
					local seed = item.jewelData and item.jewelData.conqueredBy and item.jewelData.conqueredBy.id
					if seed then
						for _, node in pairs(build.spec.nodes) do
							if node.conqueredBy and node.conqueredBy.id == seed then
								stamped = stamped + 1
							end
						end
					end
					sockets[#sockets + 1] = {
						nodeId = slot.nodeId,
						title = item.title,
						subType = item.base and item.base.subType,
						limit = item.limit,
						limitDisabled = limitDisabled,
						nodesStamped = stamped,
					}
				end
			end
		end

		entry.sockets = sockets
		entry.disabledCount = disabled
	end)
	if not ok then
		entry.error = tostring(err)
	end
	print(json.encode(entry))
end
