
local buildPath, nameFilter = arg[1], arg[2]
if not buildPath then
	io.stderr:write("REFUSING: usage: item_reroll_sweep.lua <buildXML> [modNameFilter]\n")
	os.exit(3)
end

dofile("HeadlessWrapper.lua")

local function key(m)
	local tags = {}
	for _, tag in ipairs(m) do
		tags[#tags + 1] = tostring(tag.type)
	end
	table.sort(tags)
	return string.format("%s|%s|%s|%s|%s", tostring(m.name), tostring(m.type),
		tostring(m.flags or 0), tostring(m.keywordFlags or 0), table.concat(tags, ","))
end

local function listOf(item)
	local l = item.modList
	if not l and item.slotModList then
		l = item.slotModList[1]
	end
	return l or {}
end

local function totals(item)
	local out = {}
	for _, m in ipairs(listOf(item)) do
		local v = m.value
		if type(v) == "boolean" then
			v = v and 1 or 0
		end
		if type(v) == "number" then
			local k = key(m)
			out[k] = (out[k] or 0) + v
		end
	end
	return out
end

local fh = assert(io.open(buildPath, "r"))
local xmlText = fh:read("*a")
fh:close()
loadBuildFromXML(xmlText, "reroll-sweep")
runCallback("OnFrame")

local items = build.itemsTab and build.itemsTab.items
if not items then
	io.stderr:write("REFUSING: the loaded build has no itemsTab.items\n")
	os.exit(3)
end

local scanned, skipped, diverging, rows = 0, 0, 0, 0
local ids = {}
for id in pairs(items) do
	ids[#ids + 1] = id
end
table.sort(ids)

for _, id in ipairs(ids) do
	local item = items[id]
	local raw = item and item.raw
	if type(raw) ~= "string" or raw == "" then
		skipped = skipped + 1
	else
		scanned = scanned + 1
		local ok, standalone = pcall(function() return totals(new("Item", raw)) end)
		if not ok then
			print(string.format("item %s (%s): STANDALONE PARSE FAILED: %s",
				tostring(id), tostring(item.name), tostring(standalone)))
			diverging = diverging + 1
		else
			local inBuild = totals(item)
			local keys, seen = {}, {}
			for k in pairs(standalone) do
				if not seen[k] then seen[k] = true; keys[#keys + 1] = k end
			end
			for k in pairs(inBuild) do
				if not seen[k] then seen[k] = true; keys[#keys + 1] = k end
			end
			table.sort(keys)

			local diffs = {}
			for _, k in ipairs(keys) do
				local a, b = standalone[k] or 0, inBuild[k] or 0
				if a ~= b then
					local name = k:match("^([^|]*)")
					if not nameFilter or name == nameFilter then
						diffs[#diffs + 1] = string.format("    %-52s standalone %-12s in-build %s",
							k, tostring(a), tostring(b))
					end
				end
			end
			if #diffs > 0 then
				diverging = diverging + 1
				rows = rows + #diffs
				print(string.format("item %s  %s  (rarity %s)",
					tostring(id), tostring(item.name), tostring(item.rarity)))
				for _, line in ipairs(diffs) do
					print(line)
				end
				print("  --- raw ---")
				for line in raw:gmatch("[^\r\n]+") do
					print("    " .. line)
				end
			end
		end
	end
end

print("")
print(string.format("SWEEP: %d items scanned, %d skipped (no raw), %d diverging, %d differing mod rows",
	scanned, skipped, diverging, rows))
if scanned == 0 then
	io.stderr:write("REFUSING: no item carried a raw text, so nothing was compared\n")
	os.exit(3)
end
