
local selfPath = arg[0]
local buildsDir, outDir = arg[1], arg[2]
assert(buildsDir and outDir, "usage: item_modline_order.lua <buildsDir> <outDir>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local WALK_LISTS = {
	"buffModLines",
	"enchantModLines",
	"scourgeModLines",
	"implicitModLines",
	"explicitModLines",
	"crucibleModLines",
}

local function dumpItem(item)
	local entry = {
		id = item.id,
		name = item.name,
		rarity = item.rarity,
		base = item.baseName,
		title = item.title,
		catalyst = item.catalyst,
		catalystQuality = item.catalystQuality,
		lists = {},
		walk = {},
	}
	local walkIdx = 0
	for _, listName in ipairs(WALK_LISTS) do
		local list = item[listName]
		local out = {}
		for i, modLine in ipairs(list or {}) do
			walkIdx = walkIdx + 1
			out[i] = {
				i = i,
				walkIndex = walkIdx,
				line = modLine.line,
				range = modLine.range,
				extra = type(modLine.extra) == "string" and modLine.extra or nil,
				isExtra = modLine.extra and true or false,
				modCount = modLine.modList and #modLine.modList or 0,
				crafted = modLine.crafted and true or false,
				implicit = modLine.implicit and true or false,
				variantList = modLine.variantList and (function()
					local ids = {}
					for id in pairs(modLine.variantList) do ids[#ids + 1] = id end
					table.sort(ids)
					return ids
				end)() or nil,
			}
			entry.walk[walkIdx] = { list = listName, i = i, line = modLine.line }
		end
		entry.lists[listName] = out
	end
	do
		local out = {}
		for i, modLine in ipairs(item.classRequirementModLines or {}) do
			out[i] = { i = i, line = modLine.line, range = modLine.range }
		end
		entry.lists.classRequirementModLines = out
	end
	return entry
end

lfs.mkdir(outDir)

local builds = {}
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)
assert(#builds > 0, "REFUSING: no .xml builds found under " .. buildsDir)

for _, file in ipairs(builds) do
	local path = buildsDir .. "/" .. file
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(xmlText, file)

		local itemsTab = build.itemsTab
		local items = {}
		for _, id in ipairs(itemsTab.itemOrderList or {}) do
			local item = itemsTab.items[id]
			if item then
				items[#items + 1] = dumpItem(item)
			end
		end
		entry.items = items
	end)
	if not ok then
		entry.error = tostring(err)
	end
	assert(entry.items or entry.error, "REFUSING: " .. file .. " produced neither items nor an error")

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(stable(entry), { indent = true }))
	out:close()

	io.stderr:write(string.format("modline-order: %s -> %s%s\n", file, outPath,
		entry.error and ("  ERROR " .. entry.error) or ""))
end

io.stderr:write("modline-order: wrote " .. #builds .. " builds to " .. outDir .. "\n")
