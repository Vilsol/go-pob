-- usage (cwd = <PoB-PoE2>/src): luajit poe2_save_stats.lua <build.xml> <out.json>
-- Records the numeric outputs and skill flags PoB-PoE2 calculates for a build and the PlayerStat, MinionStat and
-- FullDPSSkill rows its Save writes.
local path, outPath = arg[1], arg[2]
dofile("HeadlessWrapper.lua")
local dkjson = require "dkjson"
local f = assert(io.open(path, "rb")); local xml = f:read("*a"); f:close()
loadBuildFromXML(xml, "fixture")
runCallback("OnFrame")
local saved = build:SaveDB("fixture")
local output, flags = {}, {}
for k, v in pairs(build.calcsTab.mainOutput) do if type(v) == "number" then output[k] = v end end
for k, v in pairs(build.calcsTab.mainEnv.player.mainSkill.activeEffect.statSet.skillFlags) do if v then flags[k] = true end end
local rows = {}
for elem, stat, value in saved:gmatch('<(%w+) stat="([^"]*)" value="([^"]*)"') do rows[#rows + 1] = { elem, stat, value } end
local o = assert(io.open(outPath, "w"))
o:write(dkjson.encode({ output = output, skillFlags = flags, rows = rows }, { keyorder = {} }))
o:close()
