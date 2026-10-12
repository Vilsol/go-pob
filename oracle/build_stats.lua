
local selfPath = arg[0]
local buildsDir, outPath = arg[1], arg[2]
assert(buildsDir and outPath, "usage: build_stats.lua <buildsDir> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

local function scalars(t)
	local out = {}
	if type(t) ~= "table" then
		return out
	end
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local function modLines(lines)
	local out = {}
	for i, m in ipairs(lines or {}) do
		out[i] = {
			line = m.line,
			extra = m.extra,
			range = m.range,
			crafted = m.crafted or false,
			fractured = m.fractured or false,
		}
	end
	return out
end

local function dumpItem(item)
	local sockets = {}
	for i, s in ipairs(item.sockets or {}) do
		sockets[i] = { color = s.color, group = s.group }
	end

	return {
		raw = item.raw,
		name = item.name,
		title = item.title,
		baseName = item.baseName,
		rarity = item.rarity,
		quality = item.quality,
		itemLevel = item.itemLevel,
		corrupted = item.corrupted or false,
		fractured = item.fractured or false,
		synthesised = item.synthesised or false,
		shaper = item.shaper or false,
		elder = item.elder or false,
		requirements = scalars(item.requirements),
		sockets = sockets,
		implicitModLines = modLines(item.implicitModLines),
		explicitModLines = modLines(item.explicitModLines),
		enchantModLines = modLines(item.enchantModLines),
	}
end

local builds = {}
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		table.insert(builds, file)
	end
end
table.sort(builds)

local results = {}
for _, file in ipairs(builds) do
	local path = buildsDir .. "/" .. file
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		entry.output = scalars(build.calcsTab.mainOutput)

		local items = {}
		for i, item in ipairs(build.itemsTab.items) do
			items[i] = dumpItem(item)
		end
		entry.items = items
		entry.itemCount = #items
	end)
	if not ok then
		entry.error = tostring(err)
	end

	table.insert(results, entry)
	io.stderr:write(string.format("oracle: %s -> %s items, %s\n", file,
		tostring(entry.itemCount), entry.error and ("ERROR " .. entry.error) or "ok"))
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. #results .. " builds to " .. outPath .. "\n")
