-- usage (from PoB-PoE2's src/): luajit poe2_sidebar.lua <build.xml> <out.json>
-- Records the sidebar PoB-PoE2 shows for a build: the player's numeric and true outputs, the main
-- skill's skill flags, the stat box rows without colour codes, the warnings, and the config options the
-- config tab shows.
local selfPath, path, outPath = arg[0], arg[1], arg[2]
assert(path and outPath, "usage: poe2_sidebar.lua <build.xml> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(path))
local xml = f:read("*a")
f:close()
loadBuildFromXML(xml)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv
local output = {}
for k, v in pairs(env.player.output) do
	if type(v) == "number" or v == true then
		output[k] = v
	end
end
local flags = {}
for k, v in pairs(env.player.mainSkill.activeEffect.statSet.skillFlags) do
	if v then
		flags[k] = true
	end
end
local function plain(s)
	return (s:gsub("%^x%x%x%x%x%x%x", ""):gsub("%^%d", ""))
end
local groups, group = {}, {}
for _, row in ipairs(build.controls.statBox.list) do
	if row[1] then
		group[#group + 1] = { plain(row[1]), plain(row[2] or "") }
	elseif #group > 0 then
		groups[#groups + 1] = group
		group = {}
	end
end
if #group > 0 then
	groups[#groups + 1] = group
end

local configShown = {}
for var, control in pairs(build.configTab.varControls) do
	if control:IsShown() then
		configShown[#configShown + 1] = var
	end
end
table.sort(configShown)

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable({ output = output, skillFlags = flags, groups = groups, warnings = build.controls.warnings.lines, configShown = configShown }), { indent = true }))
o:write("\n")
o:close()
