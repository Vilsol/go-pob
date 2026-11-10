-- usage (from PoB-PoE2's src/): luajit poe2_build_planner_loadouts.lua <build.xml> <out.json>
-- Records the Import/Export tab's Export All Loadouts (ImportTab.lua:440-472) for each tree picked in the Tree
-- dropdown: BuildExportPoE2.GetLoadouts, and the file name and BuildTable WriteAllLoadouts writes for each loadout, or
-- its errors. The files go to a scratch folder and are read back.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_build_planner_loadouts.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local BuildExportPoE2 = require("Modules.BuildExportPoE2")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local dir = os.tmpname()
os.remove(dir)
assert(os.execute("mkdir -p '" .. dir .. "'") == 0)

local meta = { name = "Planner build", author = "Author", description = "", useGeneratedItemText = true }
local cases = {}
for specIndex, spec in ipairs(build.treeTab.specList) do
	-- ImportTab.lua:356-361: the save path follows the Tree dropdown's tree
	local basePath = BuildExportPoE2.BuildPath(meta.name, spec.treeVersion, dir .. "/")
	local loadouts = BuildExportPoE2.GetLoadouts(build)
	local written, errors = BuildExportPoE2.WriteAllLoadouts(build, basePath, meta, loadouts)
	local files = {}
	for _, p in ipairs(written) do
		local fh = assert(io.open(p))
		local root = json.decode(fh:read("*a"))
		fh:close()
		os.remove(p)
		for _, list in ipairs({ root.passives or {}, root.inventory_slots or {} }) do
			table.sort(list, function(a, b)
				local ka = (a.id or a.inventory_id) .. ":" .. (a.slot_x or 0)
				local kb = (b.id or b.inventory_id) .. ":" .. (b.slot_x or 0)
				return ka < kb
			end)
		end
		files[#files + 1] = { fileName = p:match("([^/\\]+)$"), build = root }
	end
	local listed = {}
	for i, l in ipairs(loadouts) do
		listed[i] = { name = l.name, fileName = l.fileName, specIndex = l.specIndex, skillSetId = l.skillSetId, itemSetId = l.itemSetId }
	end
	cases[#cases + 1] = { specIndex = specIndex, loadouts = listed, files = files, errors = errors }
end
os.execute("rmdir '" .. dir .. "'")

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ cases = cases }), { indent = true }))
o:write("\n")
o:close()
