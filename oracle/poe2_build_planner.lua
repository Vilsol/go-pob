-- usage (from PoB-PoE2's src/): luajit poe2_build_planner.lua <build.xml> <out.json>
-- Records BuildExportPoE2.BuildTable for every tree, skill set and item set of a build, with the metadata the
-- Import/Export tab's fields give, and the default name and file name for each tree.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_build_planner.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local BuildExportPoE2 = require("Modules.BuildExportPoE2")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local metas = {
	{ name = "Planner build", author = "Author", description = "", useGeneratedItemText = true },
	{ name = "Planner build", author = "Someone", description = "Line one {braces}", useGeneratedItemText = false },
}
local cases = {}
for specIndex, spec in ipairs(build.treeTab.specList) do
	for _, skillSetId in ipairs(build.skillsTab.skillSetOrderList) do
		for _, itemSetId in ipairs(build.itemsTab.itemSetOrderList) do
			for metaIndex, meta in ipairs(metas) do
				local selection = { specIndex = specIndex, skillSetId = skillSetId, itemSetId = itemSetId }
				local root = BuildExportPoE2.BuildTable(build, meta, selection)
				for _, list in ipairs({ root.passives, root.inventory_slots }) do
					table.sort(list, function(a, b)
						local ka = (a.id or a.inventory_id) .. ":" .. (a.slot_x or 0)
						local kb = (b.id or b.inventory_id) .. ":" .. (b.slot_x or 0)
						return ka < kb
					end)
				end
				cases[#cases + 1] = {
					specIndex = specIndex,
					skillSetId = skillSetId,
					itemSetId = itemSetId,
					meta = metaIndex,
					path = BuildExportPoE2.BuildPath(meta.name, spec.treeVersion, "/"):sub(2),
					build = root,
				}
			end
		end
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ cases = cases }), { indent = true }))
o:write("\n")
o:close()
